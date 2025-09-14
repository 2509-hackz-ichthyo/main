terraform {
  required_version = ">= 1.0.0"
  required_providers {
    aws = {
      source = "hashicorp/aws"
      version = "5.3.0"
    }
  }
}

provider "aws" {
  region = "ap-northeast-1"
}