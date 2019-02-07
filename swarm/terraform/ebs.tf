
# resource "aws_ebs_volume" "vol1" {
#     size = 100
#     type = "gp2"
#     availability_zone = "${aws_instance.my_instance.availability_zone}"
# }

# resource "aws_ebs_volume" "vol2" {
#     size = 500
#     type = "gp2"
#     availability_zone = "${aws_instance.my_instance.availability_zone}"
# }

# resource "aws_volume_attachment" "vol1" {
#     instance_id = "${aws_instance.my_instance.id}"
#     volume_id = "${aws_ebs_volume.vol1.id}"
#     device_name = "/dev/xvdb"
# }

# resource "aws_volume_attachment" "vol2" {
#     instance_id = "${aws_instance.my_instance.id}"
#     volume_id = "${aws_ebs_volume.vol2.id}"
#     device_name = "/dev/xvdc"
# }