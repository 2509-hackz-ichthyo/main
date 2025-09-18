# https://envader.plus/article/431
locals {
  dev = {
    system = "hackz-ichthyo-ec2"
  }
  vpc = {
    cidr_block = "10.0.0.0/16"
    subnet_cidr = "10.0.1.0/24"
  }
}

data "aws_ecr_repository" "hackz_ichthyo_ecr_repository" {
  name = "2509-hackz-ichthyo"
}

# ECS Task Execution Role
data "aws_iam_policy_document" "assume_role_policy" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

# vpc
resource "aws_vpc" "hackz_ichthyo_vpc" {
  cidr_block           = local.vpc.cidr_block
  enable_dns_support   = true
  enable_dns_hostnames = true
  tags = {
    Name  = "hackz-ichthyo-vpc"
    Roles = "vpc"
  }
}

# subnet
resource "aws_subnet" "hackz_ichthyo_subnet_public" {
  vpc_id                  = aws_vpc.hackz_ichthyo_vpc.id
  cidr_block              = local.vpc.subnet_cidr
  availability_zone       = "ap-northeast-1a"
  map_public_ip_on_launch = true
  tags = {
    Name  = "hackz-ichthyo-subnet-public"
    Roles = "subnet"
  }
}

# RouteTable
resource "aws_route_table" "hackz_ichthyo_route_table" {
  vpc_id = aws_vpc.hackz_ichthyo_vpc.id
  tags = {
    Name = "hackz-ichthyo-route-table"
  }
}

# RouteTableの関連付け
resource "aws_route_table_association" "hackz_ichthyo_public_route" {
  subnet_id      = aws_subnet.hackz_ichthyo_subnet_public.id
  route_table_id = aws_route_table.hackz_ichthyo_route_table.id
}

# InternetGateway
resource "aws_internet_gateway" "hackz_ichthyo_igw" {
  vpc_id = aws_vpc.hackz_ichthyo_vpc.id
  tags = {
    Name = "hackz-ichthyo-igw"
  }
}

# Route
resource "aws_route" "hackz_ichthyo_route" {
  route_table_id         = aws_route_table.hackz_ichthyo_route_table.id
  destination_cidr_block = "0.0.0.0/0"
  gateway_id             = aws_internet_gateway.hackz_ichthyo_igw.id
}

# SecurityGroup
resource "aws_security_group" "hackz_ichthyo_sg" {
  name        = "hackz-ichthyo-sg"
  description = "hackz-ichthyo-sg"
  vpc_id      = aws_vpc.hackz_ichthyo_vpc.id

  tags = {
    Name = "${local.dev.system}-sg"
  }
}

# Inbound/Outbound rules for fargate
resource "aws_security_group_rule" "hackz_ichthyo_rule_ingress" {
  type              = "ingress"
  from_port         = 3000
  to_port           = 3000
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.hackz_ichthyo_sg.id
}
resource "aws_security_group_rule" "hackz_ichthyo_rule_egress" {
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = -1
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.hackz_ichthyo_sg.id
}

# ECS Cluster
resource "aws_ecs_cluster" "hackz_ichthyo_ecs_cluster" {
  name = "hackz-ichthyo-ecs-cluster"
}

# ECS Task Definition
resource "aws_ecs_task_definition" "hackz_ichthyo_ecs_task_definition" {
  family                   = "hackz-ichthyo-ecs-task-definition"
  cpu                      = "256"
  memory                   = "512"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = aws_iam_role.hackz_ichthyo_ecs_task_execution_role.arn
  container_definitions = jsonencode([
    {
      name      = "hackz-ichthyo-container"
      image     = "471112951833.dkr.ecr.ap-northeast-1.amazonaws.com/2509-hackz-ichthyo:latest"
      cpu       = 256
      memory    = 512
      essential = true

      portMappings = [
        {
          containerPort = 3000
          protocol      = "tcp"
        }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = "/ecs/hackz-ichthyo-ecs"
          "awslogs-region"        = "ap-northeast-1"
          "awslogs-stream-prefix" = "ecs"
        }
      }
    }
  ])

  runtime_platform {
    cpu_architecture        = "X86_64"
    operating_system_family = "LINUX"
  }
}

# ECS Service
resource "aws_ecs_service" "hackz_ichthyo_ecs_service" {
  name            = "hackz-ichthyo-ecs-service"
  cluster         = aws_ecs_cluster.hackz_ichthyo_ecs_cluster.id
  task_definition = aws_ecs_task_definition.hackz_ichthyo_ecs_task_definition.arn
  desired_count   = 1  # Start service
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = [aws_subnet.hackz_ichthyo_subnet_public.id]
    security_groups  = [aws_security_group.hackz_ichthyo_sg.id]
    assign_public_ip = true
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.hackz_ichthyo_tg.arn
    container_name   = "hackz-ichthyo-container"
    container_port   = 3000
  }

  depends_on = [aws_lb_listener.hackz_ichthyo_listener]
}

# Task実行用 IAM Role(ECRリポジトリからイメージをpullしてくる際に必要)
resource "aws_iam_role" "hackz_ichthyo_ecs_task_execution_role" {
  name               = "hackz-ichthyo-ecs-task-execution-role"
  assume_role_policy = data.aws_iam_policy_document.assume_role_policy.json
}

# IAM Policyをロールにアタッチ
resource "aws_iam_role_policy_attachment" "hackz_ichthyo_ecs_task_execution_role" {
  role       = aws_iam_role.hackz_ichthyo_ecs_task_execution_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

# CloudWatch Log Group
resource "aws_cloudwatch_log_group" "hackz_ichthyo_log_group" {
  name              = "/ecs/hackz-ichthyo-ecs"
  retention_in_days = 30
}

# Elastic IP for Network Load Balancer
resource "aws_eip" "hackz_ichthyo_nlb_eip" {
  domain = "vpc"
  tags = {
    Name = "hackz-ichthyo-nlb-eip"
  }
}

# Network Load Balancer
resource "aws_lb" "hackz_ichthyo_nlb" {
  name               = "hackz-ichthyo-nlb"
  internal           = false
  load_balancer_type = "network"
  
  subnet_mapping {
    subnet_id     = aws_subnet.hackz_ichthyo_subnet_public.id
    allocation_id = aws_eip.hackz_ichthyo_nlb_eip.id
  }

  enable_deletion_protection = false

  tags = {
    Name = "hackz-ichthyo-nlb"
  }
}

# Target Group for ECS Service
resource "aws_lb_target_group" "hackz_ichthyo_tg" {
  name        = "hackz-ichthyo-tg"
  port        = 3000
  protocol    = "TCP"
  target_type = "ip"
  vpc_id      = aws_vpc.hackz_ichthyo_vpc.id

  health_check {
    enabled             = true
    healthy_threshold   = 2
    protocol            = "TCP"
    unhealthy_threshold = 2
  }

  tags = {
    Name = "hackz-ichthyo-target-group"
  }
}

# Load Balancer Listener
resource "aws_lb_listener" "hackz_ichthyo_listener" {
  load_balancer_arn = aws_lb.hackz_ichthyo_nlb.arn
  port              = "3000"
  protocol          = "TCP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.hackz_ichthyo_tg.arn
  }
}

# REST API Gateway for game replay functionality
resource "aws_api_gateway_rest_api" "game_replay_api" {
  name        = "game-replay-api"
  description = "REST API for accessing archived game data for replay functionality"

  endpoint_configuration {
    types = ["REGIONAL"]
  }
}

# API Gateway Resource: /replay
resource "aws_api_gateway_resource" "replay_resource" {
  rest_api_id = aws_api_gateway_rest_api.game_replay_api.id
  parent_id   = aws_api_gateway_rest_api.game_replay_api.root_resource_id
  path_part   = "replay"
}

# API Gateway Resource: /replay/random
resource "aws_api_gateway_resource" "replay_random_resource" {
  rest_api_id = aws_api_gateway_rest_api.game_replay_api.id
  parent_id   = aws_api_gateway_resource.replay_resource.id
  path_part   = "random"
}

# API Gateway Method: GET /replay/random
resource "aws_api_gateway_method" "replay_random_get" {
  rest_api_id   = aws_api_gateway_rest_api.game_replay_api.id
  resource_id   = aws_api_gateway_resource.replay_random_resource.id
  http_method   = "GET"
  authorization = "NONE"
}

# API Gateway Method: OPTIONS /replay/random (for CORS)
resource "aws_api_gateway_method" "replay_random_options" {
  rest_api_id   = aws_api_gateway_rest_api.game_replay_api.id
  resource_id   = aws_api_gateway_resource.replay_random_resource.id
  http_method   = "OPTIONS"
  authorization = "NONE"
}

# API Gateway Integration: GET /replay/random
resource "aws_api_gateway_integration" "replay_random_get_integration" {
  rest_api_id = aws_api_gateway_rest_api.game_replay_api.id
  resource_id = aws_api_gateway_resource.replay_random_resource.id
  http_method = aws_api_gateway_method.replay_random_get.http_method

  integration_http_method = "POST"
  type                   = "AWS_PROXY"
  uri                    = aws_lambda_function.game_replay_handler.invoke_arn
}

# API Gateway Integration: OPTIONS /replay/random (for CORS)
resource "aws_api_gateway_integration" "replay_random_options_integration" {
  rest_api_id = aws_api_gateway_rest_api.game_replay_api.id
  resource_id = aws_api_gateway_resource.replay_random_resource.id
  http_method = aws_api_gateway_method.replay_random_options.http_method

  type                 = "MOCK"
  passthrough_behavior = "WHEN_NO_MATCH"

  request_templates = {
    "application/json" = jsonencode({
      statusCode = 200
    })
  }
}

# API Gateway Method Response: GET /replay/random
resource "aws_api_gateway_method_response" "replay_random_get_response" {
  rest_api_id = aws_api_gateway_rest_api.game_replay_api.id
  resource_id = aws_api_gateway_resource.replay_random_resource.id
  http_method = aws_api_gateway_method.replay_random_get.http_method
  status_code = "200"

  response_parameters = {
    "method.response.header.Access-Control-Allow-Origin"  = true
    "method.response.header.Access-Control-Allow-Methods" = true
    "method.response.header.Access-Control-Allow-Headers" = true
  }
}

# API Gateway Method Response: OPTIONS /replay/random
resource "aws_api_gateway_method_response" "replay_random_options_response" {
  rest_api_id = aws_api_gateway_rest_api.game_replay_api.id
  resource_id = aws_api_gateway_resource.replay_random_resource.id
  http_method = aws_api_gateway_method.replay_random_options.http_method
  status_code = "200"

  response_parameters = {
    "method.response.header.Access-Control-Allow-Origin"  = true
    "method.response.header.Access-Control-Allow-Methods" = true
    "method.response.header.Access-Control-Allow-Headers" = true
  }
}

# API Gateway Integration Response: GET /replay/random
resource "aws_api_gateway_integration_response" "replay_random_get_integration_response" {
  rest_api_id = aws_api_gateway_rest_api.game_replay_api.id
  resource_id = aws_api_gateway_resource.replay_random_resource.id
  http_method = aws_api_gateway_method.replay_random_get.http_method
  status_code = aws_api_gateway_method_response.replay_random_get_response.status_code

  response_parameters = {
    "method.response.header.Access-Control-Allow-Origin"  = "'*'"
    "method.response.header.Access-Control-Allow-Methods" = "'GET,OPTIONS'"
    "method.response.header.Access-Control-Allow-Headers" = "'Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token'"
  }

  depends_on = [aws_api_gateway_integration.replay_random_get_integration]
}

# API Gateway Integration Response: OPTIONS /replay/random
resource "aws_api_gateway_integration_response" "replay_random_options_integration_response" {
  rest_api_id = aws_api_gateway_rest_api.game_replay_api.id
  resource_id = aws_api_gateway_resource.replay_random_resource.id
  http_method = aws_api_gateway_method.replay_random_options.http_method
  status_code = aws_api_gateway_method_response.replay_random_options_response.status_code

  response_parameters = {
    "method.response.header.Access-Control-Allow-Origin"  = "'*'"
    "method.response.header.Access-Control-Allow-Methods" = "'GET,OPTIONS'"
    "method.response.header.Access-Control-Allow-Headers" = "'Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token'"
  }

  depends_on = [aws_api_gateway_integration.replay_random_options_integration]
}

# API Gateway Deployment
resource "aws_api_gateway_deployment" "game_replay_api_deployment" {
  depends_on = [
    aws_api_gateway_integration.replay_random_get_integration,
    aws_api_gateway_integration.replay_random_options_integration
  ]

  rest_api_id = aws_api_gateway_rest_api.game_replay_api.id
  stage_name  = "prod"

  triggers = {
    redeployment = sha1(jsonencode([
      aws_api_gateway_resource.replay_resource.id,
      aws_api_gateway_resource.replay_random_resource.id,
      aws_api_gateway_method.replay_random_get.id,
      aws_api_gateway_method.replay_random_options.id,
      aws_api_gateway_integration.replay_random_get_integration.id,
      aws_api_gateway_integration.replay_random_options_integration.id,
    ]))
  }

  lifecycle {
    create_before_destroy = true
  }
}

# Lambda Permission for API Gateway
resource "aws_lambda_permission" "game_replay_api_permission" {
  statement_id  = "AllowExecutionFromAPIGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.game_replay_handler.function_name
  principal     = "apigateway.amazonaws.com"

  source_arn = "${aws_api_gateway_rest_api.game_replay_api.execution_arn}/*/*"
}

# Output the API Gateway URL
output "game_replay_api_url" {
  description = "URL for the game replay REST API"
  value       = "${aws_api_gateway_deployment.game_replay_api_deployment.invoke_url}/replay/random"
}

# Output the static IP address
output "hackz_ichthyo_static_ip" {
  description = "Static IP address for ECS Decode API"
  value       = aws_eip.hackz_ichthyo_nlb_eip.public_ip
}

# Output the NLB endpoint
output "hackz_ichthyo_nlb_endpoint" {
  description = "Network Load Balancer endpoint"
  value       = "http://${aws_eip.hackz_ichthyo_nlb_eip.public_ip}:3000"
}
