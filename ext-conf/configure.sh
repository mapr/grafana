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
# Relies on the following variables from the master configure.sh
#
# --------------------------------------------------------------------------
#  otNodesCount        - number of openTsdb servers 
#  otPort              - port number for openTsdb servers 
#  otNodesList         - list of openTsdb servers (needs to pick one to talk to)
#  gdDefaultPort       - default port number for grafan to listen to
#  gdInterfaceHostName - external interface to listen on 
# --------------------------------------------------------------------------
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

function changePort() {
 # $1 is port number
 # $2 is config file

 # Verify options
 if [ $1 -ne 0 -a -w $2 ]
   then
   # update config file
   # use sed to do the work
   sed -i 's/(http_port = )[0-9]+/\1$1/g' $2
   return 0
 else
   return 1
 fi
}


function changeInterface() {
 # $1 is the interface Ip/hostname
 # $2 is the config file


 # Verify options
 if [ ! -z $1 -a -w $2 ]
   then
   # update config file
   # use sed to do the work
   sed -i 's/(http_addr = ).*/\1$1/g' $2
   return 0
 else
   return 1
 fi
}


function setupWardenConfFileAndStart() {
  # Install warden file
  cp ${PACKAGE_WARDEN_FILE} ${MAPR_HOME}/conf/conf.d
  sleep 5
  # XXX TODO: run mapcli command in loop to wait for it to start

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
  openTsdb_port=$4

  #this needs testing was taken from an example for graphite
  curl 'http://admin:admin@${grafan_ip}:${grafana_port}/api/datasources' -X POST -H 'Content-Type: application/json;charset=UTF-8' --data-binary '{"name":"localOpenTSDB","type":"opentsdb","url":"http://${openTsdb_ip}:${openTsdb_port}","access":"proxy","isDefault":true,"database":"spyglass"}'
 
  #verify return code
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

  # Validate the arguments
  if [ ${hosts_count} -ne ${openTsdb_hosts_count} ]
    then
    return 1
  fi

  # For now pick the first one
  echo ${otArray[0]}
  
  return 0
}


## Main

# Verify the options to the script
#
# 
# Check if there are command line arguments

if [ $# -eq 5 ]
then  
  OT_NODES_COUNT=$1
  OT_PORT=$2
  OT_NODES_LIST=$3
  GRAFANA_PORT=$4
  GRAFANA_IP=$5
else
  OT_NODES_COUNT=${otNodesCount}
  OT_PORT=${otPort}
  OT_NODES_LIST=${otNodesList}
  GRAFANA_PORT=${gdDefaultPort}
  GRAFANA_IP=${gdInterfaceHostName}  
fi

OPENTSDB_HOST=`pickOpenTSDBHost ${OT_NODES_COUNT} ${OT_NODES_LIST}`

changePort ${GRAFANA_PORT} ${PACKAGE_CONFIG_FILE}
if [ $? -ne 0] 
  then
  return 1
fi

changeInterface ${GRAFANA_IP} ${PACKAGE_CONFIG_FILE}
if [ $? -ne 0] 
  then
  return 1
fi

setupWardenConfFileAndStart
if [ $? -ne 0] 
  then
  return 1
fi

setupOpenTsdbDataSource ${GRAFANA_IP} ${GRAFANA_PORT} ${OPENTSDB_HOST} ${OT_PORT}
if [ $? -ne 0] 
  then
  return 1
fi

return 0
