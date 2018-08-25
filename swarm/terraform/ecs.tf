resource "aws_ecs_cluster" "main" {
  name = "${var.resource_tag}"
}

data "template_file" "task_definition" {
  template = "${file("${path.module}/templates/task-definition.json")}"

  vars {
    image_url           = "${var.registry_url}${element(split(",", var.namespace), count.index)}${element(split(",", var.container_name), count.index)}:${element(split(",", var.version_tag), count.index)}"
    container_name      = "${element(split(",", var.container_name), count.index)}"
    log_group_region    = "${var.aws_region}"
    log_group_name      = "${element(aws_cloudwatch_log_group.app.*.name, count.index)}"
    container_p2p_port  = "${element(split(",", var.container_p2p_port), count.index)}"
    container_rpc_port  = "${element(split(",", var.container_rpc_port), count.index)}"
    container_http_port = "${element(split(",", var.container_http_port), count.index)}"
    # alb              = "${aws_alb.main.dns_name}"
    name                = "${var.env_key}"
    value               = "${var.env_value}"
  }

  count = "${length(split(",", var.container_name))}"
}

resource "aws_ecs_task_definition" "td" {
  family                = "${var.resource_tag}_${element(split(",", var.container_name), count.index)}"
  container_definitions = "${element(data.template_file.task_definition.*.rendered, count.index)}"

  volume {
    name      = "efs"
    host_path = "/mnt/efs/${element(split(",", var.container_name), count.index)}"
  }

  task_role_arn = "${aws_iam_role.application.arn}"
  count         = "${length(split(",", var.container_name))}"
}

resource "aws_ecs_service" "swarmnode" {
  name                               = "${var.resource_tag}-${element(split(",", var.container_name),0)}"
  cluster                            = "${aws_ecs_cluster.main.id}"
  task_definition                    = "${element(aws_ecs_task_definition.td.*.arn,0)}"
  desired_count                      = "${element(split(",", var.desired_count),0)}"
  deployment_minimum_healthy_percent = 0
  deployment_maximum_percent         = 100
  # iam_role                           = "${aws_iam_role.application.arn}"

  # load_balancer {
  #   target_group_arn = "${aws_alb_target_group.web.id}"
  #   container_name   = "${element(split(",", var.container_name),0)}"
  #   container_port   = "${element(split(",", var.container_port),0)}"
  # }

  # service_registries {
  #   registry_arn = "${aws_service_discovery_service.swarmnode.arn}"
  #   port = "8081"
  # }

  depends_on = [
    "aws_iam_role_policy.application",
    "aws_iam_role_policy.ec2",
    # "aws_alb_listener.front_end",
  ]
}

# resource "aws_service_discovery_public_dns_namespace" "swarmnode" {
#   name = "node.conscience.network"
#   description = "swarmnode"
# }

# resource "aws_service_discovery_service" "swarmnode" {
#   name = "${var.container_name}"
#   dns_config {
#     namespace_id = "${aws_service_discovery_public_dns_namespace.swarmnode.id}"
#     dns_records {
#       ttl = 10
#       type = "A"
#     }
#   }
#   # health_check_config {
#   #   failure_threshold = 10
#   #   resource_path = "path"
#   #   type = "HTTP"
#   # }
# }
