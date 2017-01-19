if [ -f /etc/redhat-release ]; then

cat <<-EOF   > /etc/yum.repos.d/CentOS_people_devtools.repo
# CentOS-devtools.repo
#
# The mirror system uses the connecting IP address of the client and the
# update status of each mirror to pick mirrors that are updated to and
# geographically close to the client.  You should use this for CentOS updates
# unless you are manually picking other mirrors.
#
# If the mirrorlist= does not work for you, as a fall back you can try the 
# remarked out baseurl= line instead.
#
#

[devtools]
name=Devtools
baseurl=http://people.centos.org/tru/devtoolset-3-rebuild/x86_64/RPMS
gpgcheck=0
enabled=1
EOF


yum -y install devtoolset-3-gcc-c++-4.9.2

else

apt-get -y install python-software-properties
add-apt-repository ppa:ubuntu-toolchain-r/test
apt-get update
apt-get -y install g++-4.9
fi
