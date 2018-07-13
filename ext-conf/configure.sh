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
GRAFANA_CONF_FILE="${GRAFANA_CONF_FILE:-${GRAFANA_HOME}/etc/grafana/grafana.ini}"
NEW_GRAFANA_CONF_FILE="${NEW_GRAFANA_CONF_FILE:-${GRAFANA_CONF_FILE}.progress}"
NOW=`date "+%Y%m%d_%H%M%S"`
MAPR_HOME=${MAPR_HOME:-/opt/mapr}
GRAFANA_RETRY_DELAY=24
GRAFANA_RETRY_CNT=5
LOAD_DATA_SOURCE_ONLY=0
GRAFANA_CONF_ASSUME_RUNNING_CORE=${isOnlyRoles:-0}
GRAFANA_DEFAULT_DASHBOARDS="cldb_dashboard.json node_dashboard.json volume_dashboard.json new_volume_dashboard.json"
GRAFANA_DASHBOARD_PREFIX='{ "dashboard": '
GRAFANA_DASHBOARD_POSTFIX=', "overwrite": true, "inputs": [{ "name": "DS_MAPRMONITORINGOPENTSDB", "type": "datasource", "pluginId": "opentsdb", "value": "MaprMonitoringOpenTSDB" }] }'
GRAFANA_DASHBOARD_TMP_FILE="/tmp/gf_dashboard_$$.json"
#GRAFANA_CURL_DEBUG="-v -S"
GRAFANA_CURL_DEBUG=""
WARDEN_START_KEY="service.command.start"
WARDEN_HEAPSIZE_MIN_KEY="service.heapsize.min"
WARDEN_HEAPSIZE_MAX_KEY="service.heapsize.max"
WARDEN_HEAPSIZE_PERCENT_KEY="service.heapsize.percent"

nodecount=0
nodeport=4242
grafanaport="3000"
admin_password="${GRAFANA_ADMIN_PASSWORD:-}"
admin_user="${GRAFANA_ADMIN_ID:-}"
nodelist=""
secureCluster=0
admin_pw_given=0
admin_user_given=0
switching_security_mode=0
[ -n "$admin_password" ] && admin_pw_given=1
[ -n "$admin_user" ] && admin_user_given=1

if [ -e "${MAPR_HOME}/server/common-ecosystem.sh" ]; then
    . "${MAPR_HOME}/server/common-ecosystem.sh"
else
   echo "Failed to source common-ecosystem.sh"
   exit 0
fi

INST_WARDEN_FILE="${MAPR_CONF_CONFD_DIR}/warden.grafana.conf"
PKG_WARDEN_FILE="${GRAFANA_HOME}/etc/conf/warden.grafana.conf"
#############################################################################
# Function to get the grafana admin login information
#
#############################################################################
getGrafanaLogin() {
    local user=""
    local pw=""
    user=$(grep admin_user "$GRAFANA_CONF_FILE" | cut -d'=' -f 2 | sed -e 's/ //g')
    if [ -z "$user" ]; then
        user=$GRAFANA_DEF_USER
    fi
    pw=$(grep admin_password "$GRAFANA_CONF_FILE" | cut -d'=' -f 2 | sed -e 's/ //g')
    if [ -z "$pw" ]; then
        pw=$GRAFANA_DEF_PW
    fi
    echo "$user:$pw"
}

#############################################################################
# Function to get the port grafana is configured to listen on
#
#############################################################################
getGrafanaPort() {
    local port=""
    port=$(grep http_port "$GRAFANA_CONF_FILE" | cut -d'=' -f 2 | sed -e 's/ //g')
    if [ -z "$port" ]; then
        port=$GRAFANA_DEF_PORT
    fi
    echo "$port"
}

#############################################################################
# Function to figure out if grafana is secured
#
#############################################################################
isGrafanaSecured() {
    local protocol="http"
    local prot=""
    local isSecured=1
    prot=$(grep protocol "$GRAFANA_CONF_FILE" | cut -d'=' -f 2 | sed -e 's/ //g')
    if [ -z "$prot" ]; then
        prot=$protocol
    fi
    if [ "$prot" = "https" ]; then
        isSecured=0
    elif [ "$prot" = "http" ] ; then
        isSecured=1
    fi
    return $isSecured
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
# Function to change the admin password
#
#############################################################################
function changeAdminPassword() {
    # $1 is the password
    # $2 is config file
    # Verify options
    if [ ! -z $1 -a -w $2 ]; then
        # update config file
        # use sed to do the work
        sed -i 's/[ ;]*\(admin_password = \).*/\1'$1'/g' $2
        return $?
    else
        return 1
    fi
}

#############################################################################
# Function to change the admin user
#
#############################################################################
function changeAdminUser() {
    # $1 is the user
    # $2 is config file
    # Verify options
    if [ ! -z $1 -a -w $2 ]; then
        # update config file
        # use sed to do the work
        sed -i 's/[ ;]*\(admin_user = \).*/\1'$1'/g' $2
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
# Function to enable ssl communication between grafana server and browser
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
    else
        sed -i 's/\(\;\)*\(protocol =\).*/\2 http/;s@\(cert_file =\).*@\;\1@g;s@\(cert_key =\).*@\;\1@g' $1
        rm -f ${GRAFANA_HOME}/etc/grafana/cert.pem ${GRAFANA_HOME}/etc/grafana/key.pem
    fi
    return 0
}



#############################################################################
# Function to change ownership of our files to $MAPR_USER
#
#############################################################################
function adjustOwnerShip() {
    chown -R "$MAPR_USER":"$MAPR_GROUP" "$GRAFANA_HOME"
    chmod -R o-rwx "$GRAFANA_HOME"
}


#############################################################################
# Function to extract key from warden config file
#
# Expects the following input:
# $1 = warden file to extract key from
# $2 = the key to extract
#
#############################################################################
function get_warden_value() {
    local f=$1
    local key=$2
    local val=""
    local rc=0
    if [ -f "$f" ] && [ -n "$key" ]; then
        val=$(grep "$key" "$f" | cut -d'=' -f2 | sed 's/ //g')
        rc=$?
    fi
    echo "$val"
    return $rc
}

#############################################################################
# Function to update value for  key in warden config file
#
# Expects the following input:
# $1 = warden file to update key in
# $2 = the key to update
# $3 = the value to update with
#
#############################################################################
function update_warden_value() {
    local f=$1
    local key=$2
    local value=$3

    sed -i 's/\([ ]*'"$key"'=\).*$/\1'"$value"'/' "$f"
}

#############################################################################
# Function to enable warden to manage us
#
#############################################################################
function setupWardenConfFileAndStart() {
    local rc=0
    local curr_start_cmd
    local curr_heapsize_min
    local curr_heapsize_max
    local curr_heapsize_percent
    local curr_runstate
    local pkg_start_cmd
    local pkg_heapsize_min
    local pkg_heapsize_max
    local pkg_heapsize_percent
    local newestPrevVersionFile
    local tmpWardenFile

    tmpWardenFile=$(basename $PKG_WARDEN_FILE)
    tmpWardenFile="/tmp/${tmpWardenFile}$$"

    if [ -f "$INST_WARDEN_FILE" ]; then
        curr_start_cmd=$(get_warden_value "$INST_WARDEN_FILE" "$WARDEN_START_KEY")
        curr_heapsize_min=$(get_warden_value "$INST_WARDEN_FILE" "$WARDEN_HEAPSIZE_MIN_KEY")
        curr_heapsize_max=$(get_warden_value "$INST_WARDEN_FILE" "$WARDEN_HEAPSIZE_MAX_KEY")
        curr_heapsize_percent=$(get_warden_value "$INST_WARDEN_FILE" "$WARDEN_HEAPSIZE_PERCENT_KEY")
        curr_runstate=$(get_warden_value "$INST_WARDEN_FILE" "$WARDEN_RUNSTATE_KEY")
        pkg_start_cmd=$(get_warden_value "$PKG_WARDEN_FILE" "$WARDEN_START_KEY")
        pkg_heapsize_min=$(get_warden_value "$PKG_WARDEN_FILE" "$WARDEN_HEAPSIZE_MIN_KEY")
        pkg_heapsize_max=$(get_warden_value "$PKG_WARDEN_FILE" "$WARDEN_HEAPSIZE_MAX_KEY")
        pkg_heapsize_percent=$(get_warden_value "$PKG_WARDEN_FILE" "$WARDEN_HEAPSIZE_PERCENT_KEY")

        if [ "$curr_start_cmd" != "$pkg_start_cmd" ]; then
            cp "$PKG_WARDEN_FILE" "${tmpWardenFile}"
            if [ -n "$curr_runstate" ]; then
                echo "service.runstate=$curr_runstate" >> "${tmpWardenFile}"
            fi
            if [ -n "$curr_heapsize_min" ] && [ "$curr_heapsize_min" -gt "$pkg_heapsize_min" ]; then
                update_warden_value "${tmpWardenFile}" "$WARDEN_HEAPSIZE_MIN_KEY" "$curr_heapsize_min"
            fi
            if [ -n "$curr_heapsize_max" ] && [ "$curr_heapsize_max" -gt "$pkg_heapsize_max" ]; then
                update_warden_value "${tmpWardenFile}" "$WARDEN_HEAPSIZE_MAX_KEY" "$curr_heapsize_max"
            fi
            if [ -n "$curr_heapsize_percent" ] && [ "$curr_heapsize_percent" -gt "$pkg_heapsize_percent" ]; then
                update_warden_value "${tmpWardenFile}" "$WARDEN_HEAPSIZE_PERCENT_KEY" "$curr_heapsize_percent"
            fi
            cp "${tmpWardenFile}" "$INST_WARDEN_FILE"
            rc=$?
            rm -f "${tmpWardenFile}"
        fi
    else
        if  ! [ -d "${MAPR_CONF_CONFD_DIR}" ]; then
            mkdir -p "${MAPR_CONF_CONFD_DIR}" > /dev/null 2>&1
        fi
        newestPrevVersionFile=$(ls -t1 "$PKG_WARDEN_FILE"-[0-9]* |head -n 1)
        if [ -n "$newestPrevVersionFile" ] && [ -f "$newestPrevVersionFile" ]; then
            curr_runstate=$(get_warden_value "$newestPrevVersionFile" "$WARDEN_RUNSTATE_KEY")
            cp "$PKG_WARDEN_FILE" "${tmpWardenFile}"
            if [ -n "$curr_runstate" ]; then
                echo "service.runstate=$curr_runstate" >> "${tmpWardenFile}"
            fi
            cp "${tmpWardenFile}" "$INST_WARDEN_FILE"
            rc=$?
            rm -f "${tmpWardenFile}"
        else
            cp "$PKG_WARDEN_FILE" "$INST_WARDEN_FILE"
            rc=$?
        fi
    fi
    if [ $rc -ne 0 ]; then
        logWarn "grafana - Failed to install Warden conf file for service - service will not start"
    fi
    chown $MAPR_USER:$MAPR_GROUP "$INST_WARDEN_FILE"
    return $?
}

#############################################################################
# Function to check and register port availablilty
#
#############################################################################
function registerGrafanaPort() {
    local nodeport=$1
    if checkNetworkPortAvailability $nodeport ; then
        registerNetworkPort grafana $nodeport
        if [ $? -ne 0 ]; then
            logWarn "grafana - Failed to register port"
        fi
    else
        service=$(whoHasNetworkPort $nodeport)
        if [ "$service" != "grafana" ]; then
            logWarn "grafana - port $nodeport in use by $service service"
        fi
    fi
}

#############################################################################
# Function to configure the default data source
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

    if isGrafanaSecured ; then
        protocol="https"
        #ot_protocol="https" # commented out until we support the proxy
        no_cert_ver="-k"
    fi

    # If warden isn't running, then assume we are being uninstalled
    # core-internal, mapr-cldb, mapr-fileserver, mapr-gateway, mapr-jobtracker, mapr-nfs, mapr-tasktracker, mapr-webserver, mapr-zookeeper
    # all run configure.sh -R when being uninstalled
    if ! ${MAPR_HOME}/initscripts/mapr-warden status > /dev/null 2>&1 ; then
        return 1
    fi

    login=$(getGrafanaLogin)

    while [ $count -le $GRAFANA_RETRY_CNT ]
    do
        curl -s ${no_cert_ver} "$protocol://${login}@${grafana_ip}:${grafana_port}/api/org" > /dev/null 2>&1
        is_running=$?
        if [ ${is_running} -eq 0 ]; then
            if ! curl -s ${no_cert_ver} -XGET "$protocol://${login}@${grafana_ip}:${grafana_port}/api/datasources" | \
                fgrep MaprMonitoring > /dev/null 2>&1 ; then

                curl -s ${no_cert_ver} "$protocol://${login}@${grafana_ip}:${grafana_port}/api/datasources" \
                   -X POST -H 'Content-Type: application/json;charset=UTF-8' --data-binary \
                   '{"name":"MaprMonitoringOpenTSDB","type":"opentsdb","url":"'"${ot_protocol}://${openTsdb_ip}"'","access":"proxy","isDefault":true,"database":"mapr_monitoring","jsonData":{"tsdbResolution":1,"tsdbVersion":3}}'
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

    if isGrafanaSecured ; then
        protocol="https"
        #ot_protocol="https" # commented out until we support the proxy
        no_cert_ver="-k"
    fi

    login=$(getGrafanaLogin)

    while [ $count -le $GRAFANA_RETRY_CNT ]
    do
        OUTPUT=$(curl -s ${curl_dbg} ${no_cert_ver} "$protocol://${login}@${grafana_ip}:${grafana_port}/api/dashboards/import" -X POST -H 'Content-Type: application/json;charset=UTF-8' -d @$dashboard_file 2>&1)
        if [ $? -eq 0 ]; then
            rc=0
            break
        else
            sleep $GRAFANA_RETRY_DELAY
        fi
        (( count++ ))
    done
    if [ $rc -ne 0 ]; then
        logInfo "grafana - NOTE: Failed to load dashboard - output = $OUTPUT"
    fi

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
#sets MAPR_USER/MAPR_GROUP/logfile
initCfgEnv

grafana_usage="usage: $0 [-help] [-nodeCount <cnt>] [-nodePort <port>] [-grafanaPort <port>]\n\t[-loadDataSourceOnly] [-customSecure] [-secure] [-unsecure] [-EC <commonEcoOpts>]\n\t[-password <pw>] [-R] -OT \"ip:port,ip1:port,\" "
if [ ${#} -gt 1 ]; then
    # we have arguments - run as as standalone - need to get params and
    OPTS=$(getopt -a -o chln:p:suC:G:P:O:R -l help -l nodeCount: -l nodePort: -l EC: -l OT: -l grafanaPort: -l loadDataSourceOnly -l secure -l customSecure -l unsecure -l password: -l R -- "$@")
    if [ $? != 0 ]; then
        echo -e ${grafana_usage}
        return 2 2>/dev/null || exit 2
    fi
    eval set -- "$OPTS"

    while (( $# )) ; do
        case "$1" in
            --EC|-C)
                #Parse Common options
                #Ingore ones we don't care about
                ecOpts=($2)
                shift 2
                restOpts="$@"
                eval set -- "${ecOpts[@]} --"
                while (( $# )) ; do
                    case "$1" in
                        --OT|-OT)
                            nodelist="$2"
                            shift 2;;
                        --R|-R)
                            GRAFANA_CONF_ASSUME_RUNNING_CORE=1
                            shift 1 ;;
                        --) shift
                            ;;
                        *)
                            #echo "Ignoring common option $j"
                            shift 1;;
                    esac
                done
                shift 2 
                eval set -- "$restOpts"
                ;;
            --nodeCount|-n)
                nodecount="$2";
                shift 2;;
            --OT|-O)
                nodelist="$2";
                shift 2;;
            --nodePort|-P)
                nodeport="$2";
                shift 2;;
            --grafanaPort|-G)
                grafanaport="$2";
                shift 2;;
            --password|-p)
                if [ -n "$2" ] && [ -f "$2" ]; then
                    admin_password="$(cat "$2")";
                else
                    admin_password="$2";
                fi
                admin_pw_given=1
                shift 2;;
            --customSecure|-c)
                if [ -f "$GRAFANA_HOME/etc/.not_configured_yet" ]; then
                    # grafan added after secure 5.x grafan upgraded to customSecure
                    # 6.0 cluster. Deal with this by assuming a regular --secure path
                    :
                else 
                    # this is a little tricky. It either means a simpel configure.sh -R run
                    # or it means that grafan was part of the 5.x to 6.0 upgrade
                    # At the moment grafan knows of no other security settings besides
                    # the certs used for its web browser
                    :
                fi
                secureCluster=1;
                shift 1;;
            --secure|-s)
                secureCluster=1;
                admin_user=${GRAFANA_ADMIN_ID:-mapr}
                admin_password=${GRAFANA_ADMIN_PASSWORD:-mapr}
                shift 1;;
            --unsecure|-u)
                secureCluster=0;
                admin_user=${GRAFANA_ADMIN_ID:-admin}
                admin_password=${GRAFANA_ADMIN_PASSWORD:-admin}
                shift 1;;
            --loadDataSourceOnly|-l)
                LOAD_DATA_SOURCE_ONLY=1
                shift ;;
            --R|-R)
                GRAFANA_CONF_ASSUME_RUNNING_CORE=1
                shift ;;
            --help|-h)
                echo -e ${grafana_usage}
                return 2 2>/dev/null || exit 2
                ;;
            --)
                shift
                ;;
            *)
                echo "Unknown option $1"
                echo -e ${grafana_usage}
                return 2 2>/dev/null || exit 2
                ;;
        esac
    done
else
    echo -e "${grafana_usage}"
    return 2 2>/dev/null || exit 2
fi

if [ -z "$nodelist" ]; then
    logErr "grafana - -OT is required"
    echo -e "${grafana_usage}"
    return 2 2>/dev/null || exit 2
fi

GRAFANA_IP=$(hostname -i | head -n 1 | cut -d' ' -f1)
GRAFANA_DEFAULT_DATASOURCE=`pickOpenTSDBHost ${nodecount} ${nodelist}`
if [ $? -ne 0 ]; then
    logErr "grafana - Failed to pick default data source host"
    return 2 2> /dev/null || exit 2
fi

# check to see if we have a port
if ! echo "$GRAFANA_DEFAULT_DATASOURCE" | fgrep ':' > /dev/null 2>&1 ; then
    GRAFANA_DEFAULT_DATASOURCE="$GRAFANA_DEFAULT_DATASOURCE:$nodeport"
fi

adjustOwnerShip
if [ $LOAD_DATA_SOURCE_ONLY -ne 1 ]; then
    cp -p ${GRAFANA_CONF_FILE} ${NEW_GRAFANA_CONF_FILE}
    if [ $? -ne 0 ]; then
        logErr "grafana -  Failed to create scratch config file"
        return 2 2> /dev/null || exit 2
    fi

    changePort ${grafanaport} ${NEW_GRAFANA_CONF_FILE}
    if [ $? -ne 0 ]; then
        logErr "grafana - Failed to change the port"
        return 2 2> /dev/null || exit 2
    fi
    registerGrafanaPort "$grafanaport"

    if ( isGrafanaSecured && [ "$secureCluster" -eq 0 ] ) ||
        ( ! isGrafanaSecured && [ "$secureCluster" -eq 1 ] ); then
        switching_security_mode=1
    fi
    if [ -n "$admin_password" ]; then
        if [ "$admin_pw_given" -eq 1 ] ||
            ( [ "$admin_pw_given" -eq 0 ] && [ "$switching_security_mode" -eq 1 ] ); then
            changeAdminPassword "$admin_password" ${NEW_GRAFANA_CONF_FILE}
            if [ $? -ne 0 ]; then
                logErr "grafana - Failed to change admin password"
                return 2 2> /dev/null || exit 2
            fi
        fi
    fi
    if [ -n "$admin_user" ]; then
        if [ "$admin_user_given" -eq 1 ] ||
            ( [ "$admin_user_given" -eq 0 ] && [ "$switching_security_mode" -eq 1 ] ); then
            changeAdminUser "$admin_user" ${NEW_GRAFANA_CONF_FILE}
            if [ $? -ne 0 ]; then
                logErr "grafana - Failed to change admin user"
                return 2 2> /dev/null || exit 2
            fi
        fi
    fi
    configureSslBrowsing ${NEW_GRAFANA_CONF_FILE}
    if [ $? -ne 0 ]; then
        logErr "grafana - Failed to configure ssl for grafana"
        return 2 2> /dev/null || exit 2
    fi

    #changeInterface ${GRAFANA_IP} ${NEW_GRAFANA_CONF_FILE}

    # Install new config file
    cp -p ${GRAFANA_CONF_FILE} ${GRAFANA_CONF_FILE}.${NOW}
    cp -p ${NEW_GRAFANA_CONF_FILE} ${GRAFANA_CONF_FILE}
    rm -f ${NEW_GRAFANA_CONF_FILE}
    chmod 640 ${GRAFANA_CONF_FILE}
fi

fixFluentdConf
setupWardenConfFileAndStart
if [ $? -ne 0 ]; then
    logErr "grafana - Failed to install grafana warden config file"
    return 2 2> /dev/null || exit 2
fi

if [ $LOAD_DATA_SOURCE_ONLY -eq 1 ]; then
    setupOpenTsdbDataSource ${GRAFANA_IP} ${grafanaport} ${GRAFANA_DEFAULT_DATASOURCE}
    if [ $? -ne 0 ]; then
        logInfo "grafana - NOTE: Failed to install grafana default data source config - do it manually when you run grafana"
    else
        for df in $GRAFANA_DEFAULT_DASHBOARDS; do
            DB_JSON=$( cat ${GRAFANA_HOME}/etc/conf/$df )
            echo "$GRAFANA_DASHBOARD_PREFIX" > $GRAFANA_DASHBOARD_TMP_FILE
            cat ${GRAFANA_HOME}/etc/conf/$df >> $GRAFANA_DASHBOARD_TMP_FILE
            echo "$GRAFANA_DASHBOARD_POSTFIX" >> $GRAFANA_DASHBOARD_TMP_FILE
            loadDashboard ${GRAFANA_IP} ${grafanaport} $GRAFANA_DASHBOARD_TMP_FILE
            if [ $? -ne 0 ]; then
                logInfo "grafana - NOTE: Failed to load dashboard $df - do it manually when you run grafana"
            else
                rm -f $GRAFANA_DASHBOARD_TMP_FILE
            fi
        done
    fi
fi
if [ -f "$GRAFANA_HOME/etc/.not_configured_yet" ]; then
    rm -f "$GRAFANA_HOME/etc/.not_configured_yet"
fi

true
