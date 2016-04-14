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
GRAFANA_HOME="__INSTALL__"
GRAFANA_WARDEN_FILE="${GRAFANA_HOME}/etc/conf/warden.grafana.conf"
GRAFANA_CONF_FILE="${GRAFANA_CONF_FILE:-${GRAFANA_HOME}/etc/grafana/grafana.ini}"
NEW_GRAFANA_CONF_FILE="${NEW_GRAFANA_CONF_FILE:-${GRAFANA_CONF_FILE}.progress}"
MAPR_HOME=${MAPR_HOME:-/opt/mapr}
MAPR_CONF_DIR="${MAPR_HOME}/conf/conf.d"
GRAFANA_RETRY_DELAY=24
GRAFANA_RETRY_CNT=5
LOAD_DATA_SOURCE_ONLY=0
GRAFANA_CONF_ASSUME_RUNNING_CORE=${isOnlyRoles:-0}

nodecount=""
nodeport=""
grafanaport=""
nodelist=""

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


function setupWardenConfFileAndStart() {
    # make sure warden conf directory exist
    if ! [ -d ${MAPR_CONF_DIR} ]; then
        mkdir -p ${MAPR_CONF_DIR} > /dev/null 2>&1
    fi
    # Install warden file
    cp ${GRAFANA_WARDEN_FILE} ${MAPR_CONF_DIR}
    return $?
}

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
    local rc
    grafana_ip=$1
    grafana_port=$2
    openTsdb_ip=$3
    count=1
    rc=1
    while [ $count -le $GRAFANA_RETRY_CNT ]
    do
        curl -s http://admin:admin@${grafana_ip}:${grafana_port}/api/org > /dev/null 2>&1
        is_running=$?
        if [ ${is_running} -eq 0 ]; then
            if ! curl -s -XGET 'http://admin:admin@'"${grafana_ip}":"${grafana_port}"'/api/datasources' | fgrep MaprMonitoring > /dev/null 2>&1 ; then
                curl 'http://admin:admin@'"${grafana_ip}":"${grafana_port}"'/api/datasources' -X POST -H 'Content-Type: application/json;charset=UTF-8' --data-binary '{"name":"MaprMonitoringOpenTSDB","type":"opentsdb","url":"http://'${openTsdb_ip}'","access":"proxy","isDefault":true,"database":"mapr_monitoring"}'
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

    # Validate the arguments
    if [ ${host_count} -ne ${openTsdb_hosts_count} ]; then
        return 1
    fi

    # For now pick the first one
    echo ${otArray[0]}

    return 0
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

grafana_usage="usage: $0 -nodeCount <cnt> -OT \"ip:port,ip1:port,\" -nodePort <port> -grafanaPort <port> [-loadDataSourceOnly]"
if [ ${#} -gt 1 ]; then
    # we have arguments - run as as standalone - need to get params and
    # XXX why do we need the -o to make this work?
    OPTS=`getopt -a -o h -l nodeCount: -l nodePort: -l OT: -l grafanaPort: -l loadDataSourceOnly -- "$@"`
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
            --loadDataSourceOnly)
                  LOAD_DATA_SOURCE_ONLY=1
                  shift ;;
            -h)
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

if [ -z "$nodeport" -o -z "$nodelist" -o -z "$nodecount" -o -z "$grafanaport" ]; then
    echo "${grafana_usage}"
    return 2 2>/dev/null || exit 2
fi

GRAFANA_IP=`hostname -i`
GRAFANA_DEFAULT_DATASOURCE=`pickOpenTSDBHost ${nodecount} ${nodelist}`
if [ $? -ne 0 ]; then
    echo "WARNING: Failed to pick default data source host"
    return 2 2> /dev/null || exit 2
fi

if [ $LOAD_DATA_SOURCE_ONLY -ne 1 ]; then
    cp -p ${GRAFANA_CONF_FILE} ${NEW_GRAFANA_CONF_FILE}
    if [ $? -ne 0 ]; then
        echo "WARNING: Failed to create scratch config file"
        return 2 2> /dev/null || exit 2
    fi

    changePort ${grafanaport} ${NEW_GRAFANA_CONF_FILE}
    if [ $? -ne 0 ]; then
        echo "WARNING: Failed to change the port"
        return 2 2> /dev/null || exit 2
    fi

    #changeInterface ${GRAFANA_IP} ${NEW_GRAFANA_CONF_FILE}

    # Install new config file
    cp -p ${GRAFANA_CONF_FILE} ${GRAFANA_CONF_FILE}.${NOW}
    cp -p ${NEW_GRAFANA_CONF_FILE} ${GRAFANA_CONF_FILE}
    rm -f ${NEW_GRAFANA_CONF_FILE}

fi

if [ $GRAFANA_CONF_ASSUME_RUNNING_CORE -eq 1 ]; then
    setupWardenConfFileAndStart
    if [ $? -ne 0 ]; then
        echo "WARNING: Failed to install grafana warden config file"
        return 2 2> /dev/null || exit 2
    fi
fi

if [ $GRAFANA_CONF_ASSUME_RUNNING_CORE -eq 1 -o $LOAD_DATA_SOURCE_ONLY -eq 1 ]; then
    setupOpenTsdbDataSource ${GRAFANA_IP} ${grafanaport} ${GRAFANA_DEFAULT_DATASOURCE}
    if [ $? -ne 0 ]; then
        echo "NOTE: Failed to install grafana default data source config - do it manually when you run grafana"
    fi
fi

true
