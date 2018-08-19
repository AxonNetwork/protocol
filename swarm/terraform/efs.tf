resource "aws_efs_file_system" "ecs_efs" {
  tags {
    Name = "${aws_ecs_cluster.main.name}"
  }
}

resource "aws_efs_mount_target" "ecs_efs_target" {
  count           = "${var.az_count}"
  file_system_id  = "${aws_efs_file_system.ecs_efs.id}"
  subnet_id       = "${element(aws_subnet.public.*.id, count.index)}"
  security_groups = ["${aws_security_group.ec2_sg.id}"]
}
