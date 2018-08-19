data "aws_availability_zones" "available" {}

resource "aws_vpc" "main" {
  cidr_block = "${var.cidr_block}"
  enable_dns_hostnames = "${var.enable_dns_hostnames}"
  enable_dns_support = "${var.enable_dns_support}"
  tags { Name = "${aws_ecs_cluster.main.name}" }
}

resource "aws_vpc_dhcp_options" "main" {
  domain_name = "${var.aws_region}.compute.internal"
  domain_name_servers = ["AmazonProvidedDNS"]
}

resource "aws_vpc_dhcp_options_association" "dns_resolver" {
  vpc_id          = "${aws_vpc.main.id}"
  dhcp_options_id = "${aws_vpc_dhcp_options.main.id}"
}

resource "aws_internet_gateway" "main" {
  vpc_id = "${aws_vpc.main.id}"
}

resource "aws_network_acl" "main" {
  vpc_id = "${aws_vpc.main.id}"
  subnet_ids = ["${aws_subnet.public.*.id}"]

  egress {
    protocol   = "all"
    rule_no    = 100
    action     = "allow"
    cidr_block = "0.0.0.0/0"
    from_port  = 0
    to_port    = 0
  }

  ingress {
    protocol   = "all"
    rule_no    = 100
    action     = "allow"
    cidr_block = "0.0.0.0/0"
    from_port  = 0
    to_port    = 0
  }

  tags {
    Name = "main"
  }
}

resource "aws_route_table" "public" {
  vpc_id = "${aws_vpc.main.id}"
  tags { Name = "${var.resource_tag}.public" }
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = "${aws_internet_gateway.main.id}"
  }
}

resource "aws_route_table_association" "public" {
  count = "${var.az_count}"

  subnet_id      = "${element(aws_subnet.public.*.id, count.index)}"
  route_table_id = "${aws_route_table.public.id}"
}

resource "aws_subnet" "public" {
  count             = "${var.az_count}"
  cidr_block        = "${cidrsubnet(aws_vpc.main.cidr_block, 8, count.index)}"
  availability_zone = "${data.aws_availability_zones.available.names[count.index]}"
  vpc_id            = "${aws_vpc.main.id}"
  map_public_ip_on_launch = true
  tags {
    Name = "${var.resource_tag}.public.${data.aws_availability_zones.available.names[count.index]}"
  }
}

# resource "aws_eip" "nat" {
#   count = "${var.az_count}"
#   vpc = true
# }

# resource "aws_nat_gateway" "nat" {
#   allocation_id = "${element(aws_eip.nat.*.id, count.index)}"
#   subnet_id = "${element(aws_subnet.public.*.id, count.index)}"
#   count = "${var.az_count}"
# }

# resource "aws_route_table" "private" {
#   vpc_id = "${aws_vpc.main.id}"
#   count  = "${var.az_count}"
#   tags {
#     Name = "${var.resource_tag}.private.${data.aws_availability_zones.available.names[count.index]}"
#   }
# }

# resource "aws_subnet" "private" {
#   count             = "${var.az_count}"
#   cidr_block        = "${cidrsubnet(aws_vpc.main.cidr_block, 8, count.index + length(aws_subnet.public.*.id))}"
#   availability_zone = "${data.aws_availability_zones.available.names[count.index]}"
#   vpc_id            = "${aws_vpc.main.id}"
#   tags {
#     Name = "${var.resource_tag}.private.${data.aws_availability_zones.available.names[count.index]}"
#   }
# }

# resource "aws_route_table_association" "private" {
#   subnet_id      = "${element(aws_subnet.private.*.id, count.index)}"
#   route_table_id = "${element(aws_route_table.private.*.id, count.index)}"
#   count          = "${var.az_count}"
# }

# resource "aws_route" "nat_gateway" {
#   route_table_id         = "${element(aws_route_table.private.*.id, count.index)}"
#   destination_cidr_block = "0.0.0.0/0"
#   nat_gateway_id         = "${element(aws_nat_gateway.nat.*.id, count.index)}"
#   count                  = "${var.az_count}"
#   depends_on             = ["aws_route_table.private"]
# }
