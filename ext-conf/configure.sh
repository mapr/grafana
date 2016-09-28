#!/bin/bash
# Copyright (c) 2009 & onwards. MapR Tech, Inc., All rights reserved

#############################################################################
#
# This script is sourced from the master configure.sh, this way any variables
# we need are available to us.
#
# It also means that this script should never do an exit in the case of failure
# since that would cause the master configure.sh to exit too. Simply return
# an return value if needed. Sould be 0 for the most part.
#
# configure script for grafana
#
# When called from the master installer, expect to see the following options:
# -nodeCount ${otNodesCount} -OT "${otNodesList}" -nodePort ${otPort} -grafanaPort ${gdDefaultPort}
#
# where the
#
# -nodeCount    tells you how many openTsdb servers are configured in the cluster
# -OT           tells you the list of opentTsdb servers
# -nodePort     is the port number openTsdb should be listening on
# -grafanaPort tells you the port number Grafana should be listening on
#
# The following are the functions of this script
#
# 1) Change port number
#
# 2) Interface to bind to - external
#
# 3) Install the warden conf file and wait for warden to start it
#
# 4) setup opentsdb datasource
#
# The first two needs to be done before the last one, and grafana restarted.
# The last one has to verify that grafana is running
#
# The script may have to be called in two phases
# phase 1 - change config ${GRAFANA_HOME}/etc/grafana/grafana.ini
# phase 2 (after it has been started) - setup datasource

# This gets fillled out at package time
GRAFANA_HOME="${GRAFANA_HOME:-__INSTALL__}"
GRAFANA_WARDEN_FILE="${GRAFANA_HOME}/etc/conf/warden.grafana.conf"
GRAFANA_CONF_FILE="${GRAFANA_CONF_FILE:-${GRAFANA_HOME}/etc/grafana/grafana.ini}"
NEW_GRAFANA_CONF_FILE="${NEW_GRAFANA_CONF_FILE:-${GRAFANA_CONF_FILE}.progress}"
NOW=`date "+%Y%m%d_%H%M%S"`
MAPR_HOME=${MAPR_HOME:-/opt/mapr}
MAPR_USER=${MAPR_USER:-mapr}
MAPR_GROUP=${MAPR_GROUP:-mapr}
MAPR_CONF_DIR="${MAPR_HOME}/conf/conf.d"
GRAFANA_RETRY_DELAY=24
GRAFANA_RETRY_CNT=5
LOAD_DATA_SOURCE_ONLY=0
GRAFANA_CONF_ASSUME_RUNNING_CORE=${isOnlyRoles:-0}
GRAFANA_DEFAULT_DASHBOARDS="cldb_dashboard.json node_dashboard.json volume_dashboard.json"
GRAFANA_DASHBOARD_PREFIX='{ "dashboard": '
GRAFANA_DASHBOARD_POSTFIX=', "overwrite": true, "inputs": [{ "name": "DS_MAPRMONITORINGOPENTSDB", "type": "datasource", "pluginId": "opentsdb", "value": "MaprMonitoringOpenTSDB" }] }'
GRAFANA_DASHBOARD_TMP_FILE="/tmp/gf_dashboard_$$.json"
#GRAFANA_CURL_DEBUG="-v -S"
GRAFANA_CURL_DEBUG=""

nodecount=0
nodeport=4242
grafanaport="3000"
nodelist=""
secureCluster=0
# isSecure is set in server/configure.sh
if [ -n "$isSecure" ]; then
    if [ "$isSecure" == "true" ]; then
        secureCluster=1
    fi
fi

#############################################################################
# Function to log messages
#
# if $logFile is set the message gets logged there too
#
#############################################################################
function logMsg() {
    local msg
    msg="$(date): $1"
    echo $msg
    if [ -n "$logFile" ] ; then
        echo $msg >> $logFile
    fi
}

#############################################################################
# Function to change the port number configuration
# 
#############################################################################
function changePort() {
    # $1 is port number
    # $2 is config file
    # Verify options
    if [ ! -z $1 -a -w $2 ]; then
        # update config file
        # use sed to do the work
        sed -i 's/;\(http_port = \).*/\1'$1'/g' $2
        return $?
    else
        return 1
    fi
}


#############################################################################
# Function to change the interface configuration
# 
#############################################################################
function changeInterface() {
    # $1 is the interface Ip/hostname
    # $2 is the config file
    # Verify options
    if [ ! -z $1 -a -w $2 ]; then
        # update config file
        # use sed to do the work
        sed -i 's/\(;http_addr = \).*/\1$1/g' $2
        return $?
    else
        return 1
    fi
}

#############################################################################
# Function to enable ssl communication between kibana server and browser
# 
#############################################################################
function configureSslBrowsing() {
    # $1 is the config file

    if [ $secureCluster -eq 1 ]; then
        ${GRAFANA_HOME}/bin/export_cert.sh ${GRAFANA_HOME}/etc/grafana
        if [ $? -eq 0 ]; then
            sed -i 's/\(\;\)*\(protocol =\).*/\2 https/;s@\(\;\)*\(cert_file =\).*@\2'" ${GRAFANA_HOME}/etc/grafana/cert.pem"'@g;s@\(\;\)*\(cert_key =\).*@\2'" ${GRAFANA_HOME}/etc/grafana/key.pem"'@g' $1
            if [ $? -ne 0 ]; then
                return 1
            fi
        else
            return 1
        fi
    fi
    return 0
}



#############################################################################
# Function to enable warden to manage us
# 
#############################################################################
function setupWardenConfFileAndStart() {
    # make sure warden conf directory exist
    if ! [ -d ${MAPR_CONF_DIR} ]; then
        mkdir -p ${MAPR_CONF_DIR} > /dev/null 2>&1
    fi
    # Install warden file
    cp ${GRAFANA_WARDEN_FILE} ${MAPR_CONF_DIR}
    chown ${MAPR_USER}:${MAPR_GROUP} ${MAPR_CONF_DIR}/warden.grafana.conf
    return $?
}

#############################################################################
# Function to configure teh defautl data source
# 
#############################################################################
function setupOpenTsdbDataSource() {
    # $1 is the interface Ip/hostname for grafana
    # $2 is the port number for grafana
    # $3 is the interface of the opentsdb server
    # $4 is the port number of the opentsdb server
    # Verify options
    local grafana_ip
    local grafana_port
    local openTsdb_ip
    local count
    local protocol="http"
    local ot_protocol="http"
    local no_cert_ver=""
    local rc
    grafana_ip=$1
    grafana_port=$2
    openTsdb_ip=$3
    count=1
    rc=1
    curl_dbg="$GRAFANA_CURL_DEBUG"

    if [ $secureCluster -eq 1 ]; then
        protocol="https"
        #ot_protocol="https" # commented out until we support the proxy
        no_cert_ver="-k"
    fi

    # If warden isn't running, then assume we are being uninstalled
    # core-internal, mapr-cldb, mapr-fileserver, mapr-gateway, mapr-jobtracker, mapr-nfs, mapr-tasktracker, mapr-webserver, mapr-zookeeper
    # all run configure.sh -R when being uninstalled
    if ! ${MAPR_HOME}/initscripts/mapr-warden status > /dev/null 2>&1 ; then
        return 0
    fi
    while [ $count -le $GRAFANA_RETRY_CNT ]
    do
        curl -s ${no_cert_ver} "$protocol://admin:admin@${grafana_ip}:${grafana_port}/api/org" > /dev/null 2>&1
        is_running=$?
        if [ ${is_running} -eq 0 ]; then
            if ! curl -s ${no_cert_ver} -XGET "$protocol://admin:admin@${grafana_ip}:${grafana_port}/api/datasources" | fgrep MaprMonitoring > /dev/null 2>&1 ; then
                curl ${no_cert_ver} "$protocol://admin:admin@${grafana_ip}:${grafana_port}/api/datasources" -X POST -H 'Content-Type: application/json;charset=UTF-8' --data-binary '{"name":"MaprMonitoringOpenTSDB","type":"opentsdb","url":"'"${ot_protocol}://${openTsdb_ip}"'","access":"proxy","isDefault":true,"database":"mapr_monitoring"}'
                if [ $? -eq 0 ]; then
                    rc=0
                    break
                fi
            else
                rc=0
                break
            fi
        else
            sleep $GRAFANA_RETRY_DELAY
        fi
        (( count++ ))
    done

    return $rc
}

#############################################################################
# Function to load a  dashboard
# 
#############################################################################
function loadDashboard() {
    # $1 is the interface Ip/hostname for grafana
    # $2 is the port number for grafana
    # $3 is the json file to load

    local grafana_ip
    local grafana_port
    local protocol="http"
    local no_cert_ver=""
    local count
    local rc
    grafana_ip=$1
    grafana_port=$2
    dashboard_file=$3
    curl_dbg="$GRAFANA_CURL_DEBUG"
    count=1
    rc=1

    if [ $secureCluster -eq 1 ]; then
        protocol="https"
        #ot_protocol="https" # commented out until we support the proxy
        no_cert_ver="-k"
    fi
    while [ $count -le $GRAFANA_RETRY_CNT ]
    do
        curl ${curl_dbg} ${no_cert_ver} "$protocol://admin:admin@${grafana_ip}:${grafana_port}/api/dashboards/import" -X POST -H 'Content-Type: application/json;charset=UTF-8' -d @$dashboard_file
        if [ $? -eq 0 ]; then
            rc=0
            break
        else
            sleep $GRAFANA_RETRY_DELAY
        fi
        (( count++ ))
    done

    return $rc
}

#############################################################################
# Function to pick the opneTsdb host to connect to
# 
#############################################################################
function pickOpenTSDBHost() {
    # $1 is opentsdb nodes count
    # $2 is opentsdb nodes list

    # Verify options
    local openTsdb_hosts_count=$1
    local openTsdb_hosts=$2
    local host_count=0

    IFS=',' read -r -a otArray <<< "$2"
    host_count=${#otArray[@]}
    if [ $host_count -eq 0 ]; then
        return 1
    fi

    # Validate the arguments only if we were given a nodecount to validate against
    if [ ${openTsdb_hosts_count} -gt 0 -a ${host_count} -ne ${openTsdb_hosts_count} ]; then
        return 1
    fi

    # For now pick the first one
    echo ${otArray[0]}

    return 0
}

#############################################################################
# Function to fix fluentd config file
# 
#############################################################################
function fixFluentdConf() {

    cat << 'EOF' > /tmp/fix_grafana_$$.awk
/^# grafana/		{ found_gf_section=1; print; next}
/^# 3.1.1/ 		{ if (found_gf_section == 1 ) found_gf_section=0 }
/^#/      		{ if (found_gf_section == 1 ) {print "# 3.1.1 format";
                          print "#t=2016-09-21T17:35:27-0700 lvl=eror msg=\"Request Completed\" logger=context userId=1 orgId=1 uname=admin method=POST path=/api/dashboards/import status=500 remote_addr=10.10.10.73 time_ns=53ns size=0"; next }}
/ format_firstline/     { if (found_gf_section == 1) {print "  format_firstline /^t=\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}/"; next }}
/ format1/              { if (found_gf_section == 1 ) {print "  format1 /^t=(?<my_event_time>[^ ]*) lvl=(?<level>[^ ]*) msg=(?<message>.*)$/";  found_gf_section=0; next }}
                        { print }
EOF
   
    FD_CONF=(/opt/mapr/fluentd/fluentd*/etc/fluentd/fluentd.conf)
    for fd_f in $FD_CONF ; do
        if [ -f "$fd_f" ]; then
            fd_f_gf_sv="$(dirname $fd_f)/fluentd.conf.grafana_sv"
            if [ ! -f "$fd_f_gf_sv" ]; then
                cp "$fd_f" "$fd_f_gf_sv"
            fi
            cat $fd_f | awk -f /tmp/fix_grafana_$$.awk > /tmp/fd.conf
            if [ $? -eq 0 ]; then
                mv /tmp/fd.conf $fd_f
                rm -f /tmp/fd.conf /tmp/fix_grafana_$$.awk
            fi
        fi
    done
    
}


## Main

# typically called from master configure.sh with the following arguments
#
# configure.sh  -nodeCount ${otNodesCount} -OT "${otNodesList}"
#               -nodePort ${otPort} -grafanaPort $gdPort
#
# we need will use the roles file to know if this node is a RM. If this RM
# is not the active one, we will be getting 0s for the stats.
#

grafana_usage="usage: $0 [-nodeCount <cnt>] [-nodePort <port>] [-grafanaPort <port>] [-secureCluster] [-loadDataSourceOnly] [-R] -OT \"ip:port,ip1:port,\" "
if [ ${#} -gt 1 ]; then
    # we have arguments - run as as standalone - need to get params and
    # XXX why do we need the -o to make this work?
    OPTS=`getopt -a -o h -l nodeCount: -l nodePort: -l OT: -l grafanaPort: -l secureCluster -l loadDataSourceOnly -l R -- "$@"`
    if [ $? != 0 ]; then
        echo ${grafana_usage}
        return 2 2>/dev/null || exit 2
    fi
    eval set -- "$OPTS"

    for i ; do
        case "$i" in
            --nodeCount)
                  nodecount="$2";
                  shift 2;;
            --OT)
                  nodelist="$2";
                  shift 2;;
            --nodePort)
                  nodeport="$2";
                  shift 2;;
            --grafanaPort)
                  grafanaport="$2";
                  shift 2;;
            --secureCluster)
                  secureCluster=1;
                  shift 1;;
            --loadDataSourceOnly)
                  LOAD_DATA_SOURCE_ONLY=1
                  shift ;;
            --R)
                  GRAFANA_CONF_ASSUME_RUNNING_CORE=1
                  shift ;;
            --h)
                  echo ${grafana_usage}
                  return 2 2>/dev/null || exit 2
                  ;;
            --)
                  shift;;
        esac
    done

else
    echo "${grafana_usage}"
    return 2 2>/dev/null || exit 2
fi

if [ -z "$nodelist" ]; then
    logMsg "-OT is required"
    echo "${grafana_usage}"
    return 2 2>/dev/null || exit 2
fi

GRAFANA_IP=`hostname -i`
GRAFANA_DEFAULT_DATASOURCE=`pickOpenTSDBHost ${nodecount} ${nodelist}`
if [ $? -ne 0 ]; then
    logMsg "ERROR: Failed to pick default data source host"
    return 2 2> /dev/null || exit 2
fi

# check to see if we have a port
if ! echo "$GRAFANA_DEFAULT_DATASOURCE" | fgrep ':' > /dev/null 2>&1 ; then
    GRAFANA_DEFAULT_DATASOURCE="$GRAFANA_DEFAULT_DATASOURCE:$nodeport"
fi

if [ $LOAD_DATA_SOURCE_ONLY -ne 1 ]; then
    cp -p ${GRAFANA_CONF_FILE} ${NEW_GRAFANA_CONF_FILE}
    if [ $? -ne 0 ]; then
        logMsg "ERROR: Failed to create scratch config file"
        return 2 2> /dev/null || exit 2
    fi

    changePort ${grafanaport} ${NEW_GRAFANA_CONF_FILE}
    if [ $? -ne 0 ]; then
        logMsg "ERROR: Failed to change the port"
        return 2 2> /dev/null || exit 2
    fi

    configureSslBrowsing ${NEW_GRAFANA_CONF_FILE}
    if [ $? -ne 0 ]; then
        logMsg "ERROR: Failed to configure ssl for grafana"
        return 2 2> /dev/null || exit 2
    fi

    #changeInterface ${GRAFANA_IP} ${NEW_GRAFANA_CONF_FILE}

    # Install new config file
    cp -p ${GRAFANA_CONF_FILE} ${GRAFANA_CONF_FILE}.${NOW}
    cp -p ${NEW_GRAFANA_CONF_FILE} ${GRAFANA_CONF_FILE}
    rm -f ${NEW_GRAFANA_CONF_FILE}

fi

fixFluentdConf

if [ $GRAFANA_CONF_ASSUME_RUNNING_CORE -eq 1 ]; then
    setupWardenConfFileAndStart
    if [ $? -ne 0 ]; then
        logMsg "ERROR: Failed to install grafana warden config file"
        return 2 2> /dev/null || exit 2
    fi
fi

if [ $GRAFANA_CONF_ASSUME_RUNNING_CORE -eq 1 -o $LOAD_DATA_SOURCE_ONLY -eq 1 ]; then
    setupOpenTsdbDataSource ${GRAFANA_IP} ${grafanaport} ${GRAFANA_DEFAULT_DATASOURCE}
    if [ $? -ne 0 ]; then
        logMsg "NOTE: Failed to install grafana default data source config - do it manually when you run grafana"
    else
        for df in $GRAFANA_DEFAULT_DASHBOARDS; do
            DB_JSON=$( cat ${GRAFANA_HOME}/etc/conf/$df )
            echo "$GRAFANA_DASHBOARD_PREFIX" > $GRAFANA_DASHBOARD_TMP_FILE
            cat ${GRAFANA_HOME}/etc/conf/$df >> $GRAFANA_DASHBOARD_TMP_FILE
            echo "$GRAFANA_DASHBOARD_POSTFIX" >> $GRAFANA_DASHBOARD_TMP_FILE
            loadDashboard ${GRAFANA_IP} ${grafanaport} $GRAFANA_DASHBOARD_TMP_FILE
            if [ $? -ne 0 ]; then
                logMsg "NOTE: Failed to load dashboard $df - do it manually when you run grafana"
            else
                rm -f $GRAFANA_DASHBOARD_TMP_FILE
            fi
        done
    fi
fi

true
