# resource "aws_security_group" "lb_sg" {
#   description = "controls access to the application ELB"

#   vpc_id = "${aws_vpc.main.id}"
#   name   = "${var.resource_tag}-lb"

#   ingress {
#     protocol    = "tcp"
#     from_port   = 80
#     to_port     = 80
#     cidr_blocks = ["0.0.0.0/0"]
#   }

#   egress {
#     from_port   = 0
#     to_port     = 0
#     protocol    = "-1"
#     cidr_blocks = ["0.0.0.0/0"] # should be ec2_sg.id, circular dependency
#   }

#   tags {
#     key   = "Name"
#     value = "${var.resource_tag}-lb"
#   }
# }

resource "aws_security_group" "ec2_sg" {
  description = "controls direct access to application instances"
  vpc_id      = "${aws_vpc.main.id}"
  name        = "${var.resource_tag}-ec2"

  # P2P port
  ingress {
    protocol  = "tcp"
    from_port = 1337
    to_port   = 1337
    cidr_blocks = [
      "0.0.0.0/0",
    ]
  }

  # RPC port
  ingress {
    protocol  = "tcp"
    from_port = 1338
    to_port   = 1338
    cidr_blocks = [
      "0.0.0.0/0",
    ]
  }

  # HTTP port
  ingress {
    protocol  = "tcp"
    from_port = 8081
    to_port   = 8081
    cidr_blocks = [
      "0.0.0.0/0",
    ]
  }

  # SSH
  ingress {
    protocol  = "tcp"
    from_port = 22
    to_port   = 22

    cidr_blocks = [
      "0.0.0.0/0",
    ]
  }

  # NFS (for EFS)
  ingress {
    protocol  = "tcp"
    from_port = 2049
    to_port   = 2049
    self      = true
  }

  ingress {
    protocol  = "tcp"
    from_port = 9991
    to_port   = 9991
    cidr_blocks = [
      "0.0.0.0/0",
    ]
  }

  ingress {
    protocol  = "tcp"
    from_port = 6060
    to_port   = 6060
    cidr_blocks = [
      "0.0.0.0/0",
    ]
  }

  # ingress {
  #   protocol  = "tcp"
  #   from_port = 1337
  #   to_port   = 1337

  #   # Dynamic Port Range for Application Load Balancer
  #   security_groups = ["${aws_security_group.lb_sg.id}"]
  # }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    key   = "Name"
    value = "${var.resource_tag}-ec2"
  }
}

# variable "admin_cidr_ingress" {
#   description = "Open SSH on instances to only this IP range"
# }

variable "key_name" {
  description = "Key in AWS you would like to use to access you instances"
}
