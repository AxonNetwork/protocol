

# resource "aws_route53_record" "node" {
#   zone_id = "${aws_route53_zone.primary.zone_id}"
#   name    = "node.conscience.network"
#   type    = "A"
#   ttl     = "300"
#   records = ["${aws_eip.lb.public_ip}"]
# }