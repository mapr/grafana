if [ -f /etc/redhat-release ]; then
    yum -y install centos-release-scl
    yum -y install devtoolset-4
else
    apt-get -y install python-software-properties
    add-apt-repository ppa:ubuntu-toolchain-r/test
    apt-get update
    apt-get -y install g++-4.9
fi
