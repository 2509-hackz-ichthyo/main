# https://envader.plus/article/431

terraform {
  required_version = ">= 1.0.0"
  required_providers {
    aws = {
      source = "hashicorp/aws"
      version = "5.3.0"
    }
    archive = {
      source  = "hashicorp/archive"
      version = "~> 2.2"
    }
  }
}

provider "aws" {
  region = "ap-northeast-1"
}

# Data sources for region and account ID
data "aws_region" "current" {}
data "aws_caller_identity" "current" {}