#!/bin/bash

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
# phase 1 - change config ${PACKAGE_INSTALL_DIR}/etc/grafana/grafana.ini
# phase 2 (after it has been started) - setup datasource

# This gets fillled out at package time
PACKAGE_INSTALL_DIR="__INSTALL__"
PACKAGE_WARDEN_FILE="${PACKAGE_INSTALL_DIR}/etc/conf/warden.grafana.conf"
PACKAGE_CONFIG_FILE="${PACKAGE_INSTALL_DIR}/etc/grafana/grafana.ini"
MAPR_HOME=${MAPR_HOME:-/opt/mapr}
MAPR_CONF_DIR="${MAPR_HOME}/conf/conf.d"
GRAFANA_RETRY_DELAY=15
LOAD_DATA_SOURCE_ONLY=0

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
   return 0
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
   return 0
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
  cp ${PACKAGE_WARDEN_FILE} ${MAPR_CONF_DIR}

  return 0
}

function setupOpenTsdbDataSource() {
  # $1 is the interface Ip/hostname for grafana
  # $2 is the port number for grafana
  # $3 is the interface of the opentsdb server
  # $4 is the port number of the opentsdb server
  # Verify options
  grafana_ip=$1
  grafana_port=$2
  openTsdb_ip=$3
  count=1
  while [ $count -le 5 ]
  do
    curl http://admin:admin@${grafana_ip}:${grafana_port}/api/org > /dev/null 2>&1
    is_running=$?
    if [ ${is_running} -eq 0 ]; then 
      curl 'http://admin:admin@'"${grafana_ip}":"${grafana_port}"'/api/datasources' -X POST -H 'Content-Type: application/json;charset=UTF-8' --data-binary '{"name":"MaprMonitoringOpenTSDB","type":"opentsdb","url":"http://'${openTsdb_ip}'","access":"proxy","isDefault":true,"database":"mapr_monitoring"}'
      if [ $? -eq 0 ]; then 
        break
      fi 
    else
      sleep $GRAFANA_RETRY_DELAY
    fi
    (( count++ ))
  done
  
  return 0
}

function pickOpenTSDBHost() {
  # $1 is opentsdb nodes count
  # $2 is opentsdb nodes list
 
  # Verify options
  openTsdb_hosts_count=$1
  openTsdb_hosts=$2

  IFS=',' read -r -a otArray <<< "$2"
  hosts_count=${#otArray[@]}
  if [ -z "$host_count" ]; then
    return 1
  fi

  # Validate the arguments
  if [ ${hosts_count} -ne ${openTsdb_hosts_count} ]; then
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

usage="usage: $0 -nodeCount <cnt> -OT \"ip:port,ip1:port,\" -nodePort <port> -grafanaPort <port> [-loadDataSourceOnly]"
if [ ${#} -gt 1 ] ; then
   # we have arguments - run as as standalone - need to get params and
   # XXX why do we need the -o to make this work?
   OPTS=`getopt -a -o h -l nodeCount: -l nodePort: -l OT: -l grafanaPort: -l loadDataSourceOnly -- "$@"`
   if [ $? != 0 ] ; then
      echo ${usage}
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
              echo ${usage}
              return 2 2>/dev/null || exit 2
              ;;
         --)
              shift;;
      esac
   done

else
   echo "${usage}"
   return 2 2>/dev/null || exit 2
fi

if [ -z "$nodeport" -o -z "$nodelist" -o -z "$nodecount" -o -z "$grafanaport" ] ; then
    echo "${usage}"
   return 2 2>/dev/null || exit 2
fi

GRAFANA_IP=`hostname -I`
OPENTSDB_HOST=`pickOpenTSDBHost ${nodecount} ${nodelist}`
if [ $LOAD_DATA_SOURCE_ONLY -ne 1 ]; then
    if [ $? -ne 0 ]; then
      return 2 2> /dev/null || exit 2
    fi
    
    changePort ${grafanaport} ${PACKAGE_CONFIG_FILE}
    if [ $? -ne 0 ]; then 
      return 2 2> /dev/null || exit 2
    fi
    
    #changeInterface ${GRAFANA_IP} ${PACKAGE_CONFIG_FILE}
    setupWardenConfFileAndStart
    if [ $? -ne 0 ]; then
      return 2 2> /dev/null || exit 2
    fi
fi

setupOpenTsdbDataSource ${GRAFANA_IP} ${grafanaport} ${OPENTSDB_HOST}
if [ $? -ne 0 ]; then
  return 2 2> /dev/null || exit 2
fi

true
