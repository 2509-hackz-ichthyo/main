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
  name = "hackz-ichthyo"
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
resource "aws_vpc" "dev_vpc" {
  cidr_block           = local.vpc.cidr_block
  enable_dns_support   = true
  enable_dns_hostnames = true
  tags = {
    Name  = "hackz-ichthyo-vpc"
    Roles = "vpc"
  }
}

# subnet
resource "aws_subnet" "hackz-ichthyo_subnet_public" {
  vpc_id                  = aws_vpc.dev_vpc.id
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
  vpc_id = aws_vpc.dev_vpc.id
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
      image     = "471112951833.dkr.ecr.ap-northeast-1.amazonaws.com/ichthyo:1.0.0"
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
    cpu_architecture        = "ARM64"
    operating_system_family = "LINUX"
  }
}

# ECS Service
resource "aws_ecs_service" "hackz_ichthyo_ecs_service" {
  name            = "hackz-ichthyo-ecs-service"
  cluster         = aws_ecs_cluster.hackz_ichthyo_ecs_cluster.id
  task_definition = aws_ecs_task_definition.hackz_ichthyo_ecs_task_definition.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = [aws_subnet.hackz_ichthyo_subnet_public.id]
    security_groups  = [aws_security_group.hackz_ichthyo_sg.id]
    assign_public_ip = true
  }
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

