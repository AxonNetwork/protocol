#!/bin/bash


###########################################################
## Global init
###########################################################


#Join the default ECS cluster
echo ECS_CLUSTER=${ecs_cluster} >> /etc/ecs/ecs.config
PATH=$PATH:/usr/local/bin
#Instance should be added to an security group that allows HTTP outbound
yum update




###########################################################
## Set up EBS mount via Cloudstor Docker volume plugin
###########################################################


# Updating the ECS agent
yum update -y ecs-init

# Use this command if you want to support EBS as well as EFS
#docker plugin install --alias cloudstor:aws --grantall-permissions docker4x/cloudstor:18.03.0-ce-aws1 CLOUD_PLATFORM=AWS EFS_ID_REGULAR=<YOUR_EFS_ID> EFS_ID_MAXIO=<YOUR_MAXIO_EFS_ID> AWS_REGION=<REGION_NAME EFS_SUPPORTED=1 DEBUG=1

# Use this command if you only want to support EBS
docker plugin install --alias cloudstor:aws --grant-all-permissions docker4x/cloudstor:18.03.0-ce-aws1 CLOUD_PLATFORM=AWS AWS_REGION=${aws_region} EFS_SUPPORTED=0 DEBUG=1

mkdir /dev/mqueue



###########################################################
## Set up AWS EFS mount
###########################################################

#Install NFS client
if ! rpm -qa | grep -qw nfs-utils; then
    yum -y install nfs-utils
fi

#Get region of EC2 from instance metadata
EC2_AVAIL_ZONE=$(curl -L http://169.254.169.254/latest/meta-data/placement/availability-zone)
#Create mount point
DIR_TGT=/mnt/efs/
if [ ! -d "$DIR_TGT" ]; then
  mkdir -p $DIR_TGT
fi

#Instance needs to be a member of security group that allows 2049 inbound/outbound
#The security group that the instance belongs to has to be added to EFS file system configuration
#Create variables for source and target
DIR_SRC=$${EC2_AVAIL_ZONE}.${efs_id}.efs.${aws_region}.amazonaws.com
#Mount EFS file system
mount -t nfs4 $DIR_SRC:/ $DIR_TGT
#Backup fstab
cp -p /etc/fstab /etc/fstab.back-$(date +%F)
#Append line to fstab
echo -e "$DIR_SRC:/ \t\t $DIR_TGT \t\t nfs \t\t defaults \t\t 0 \t\t 0" | tee -a /etc/fstab

# ECS-Optimized AMI filesystem mount will not propagate to the Docker daemon until it's restarted
# because the Docker daemon's mount namespace is unshared from the host's at launch.
service docker restart


