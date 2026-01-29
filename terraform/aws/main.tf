# ebot AWS Infrastructure
# Complete Terraform configuration for deploying ebot on AWS

terraform {
  required_version = ">= 1.0"
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# VPC Configuration
resource "aws_vpc" "ebot" {
  cidr_block           = var.vpc_cidr
  enable_dns_hostnames = true
  enable_dns_support   = true
  
  tags = {
    Name        = "ebot-vpc-${var.environment}"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# Subnets
resource "aws_subnet" "public" {
  count             = 3
  vpc_id            = aws_vpc.ebot.id
  cidr_block        = cidrsubnet(var.vpc_cidr, 8, count.index)
  availability_zone = data.aws_availability_zones.available.names[count.index]
  
  map_public_ip_on_launch = true
  
  tags = {
    Name = "ebot-public-${count.index + 1}"
  }
}

resource "aws_subnet" "private" {
  count             = 3
  vpc_id            = aws_vpc.ebot.id
  cidr_block        = cidrsubnet(var.vpc_cidr, 8, count.index + 10)
  availability_zone = data.aws_availability_zones.available.names[count.index]
  
  tags = {
    Name = "ebot-private-${count.index + 1}"
  }
}

# Internet Gateway
resource "aws_internet_gateway" "ebot" {
  vpc_id = aws_vpc.ebot.id
  
  tags = {
    Name = "ebot-igw"
  }
}

# EKS Cluster
resource "aws_eks_cluster" "ebot" {
  name     = "ebot-${var.environment}"
  role_arn = aws_iam_role.eks_cluster.arn
  version  = var.kubernetes_version
  
  vpc_config {
    subnet_ids = concat(
      aws_subnet.public[*].id,
      aws_subnet.private[*].id
    )
    endpoint_private_access = true
    endpoint_public_access  = true
  }
  
  enabled_cluster_log_types = [
    "api",
    "audit",
    "authenticator"
  ]
  
  depends_on = [
    aws_iam_role_policy_attachment.eks_cluster_policy
  ]
}

# RDS PostgreSQL
resource "aws_db_instance" "ebot" {
  identifier        = "ebot-${var.environment}"
  engine            = "postgres"
  engine_version    = "15.4"
  instance_class    = var.db_instance_class
  allocated_storage = var.db_allocated_storage
  
  db_name  = "ebot"
  username = var.db_username
  password = var.db_password
  
  vpc_security_group_ids = [aws_security_group.rds.id]
  db_subnet_group_name   = aws_db_subnet_group.ebot.name
  
  backup_retention_period = 7
  backup_window          = "03:00-04:00"
  maintenance_window     = "Mon:04:00-Mon:05:00"
  
  multi_az               = var.environment == "production"
  storage_encrypted      = true
  deletion_protection    = var.environment == "production"
  
  tags = {
    Name        = "ebot-db-${var.environment}"
    Environment = var.environment
  }
}

# ElastiCache Redis
resource "aws_elasticache_cluster" "ebot" {
  cluster_id           = "ebot-${var.environment}"
  engine               = "redis"
  engine_version       = "7.0"
  node_type            = var.redis_node_type
  num_cache_nodes      = 1
  parameter_group_name = "default.redis7"
  port                 = 6379
  
  subnet_group_name  = aws_elasticache_subnet_group.ebot.name
  security_group_ids = [aws_security_group.redis.id]
  
  tags = {
    Name = "ebot-redis-${var.environment}"
  }
}

# S3 Bucket for storage
resource "aws_s3_bucket" "ebot" {
  bucket = "ebot-${var.environment}-${data.aws_caller_identity.current.account_id}"
  
  tags = {
    Name        = "ebot-storage-${var.environment}"
    Environment = var.environment
  }
}

resource "aws_s3_bucket_versioning" "ebot" {
  bucket = aws_s3_bucket.ebot.id
  
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "ebot" {
  bucket = aws_s3_bucket.ebot.id
  
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# Variables
variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Environment (development, staging, production)"
  type        = string
}

variable "vpc_cidr" {
  description = "VPC CIDR block"
  type        = string
  default     = "10.0.0.0/16"
}

variable "kubernetes_version" {
  description = "Kubernetes version"
  type        = string
  default     = "1.28"
}

variable "db_instance_class" {
  description = "RDS instance class"
  type        = string
  default     = "db.t3.medium"
}

variable "db_allocated_storage" {
  description = "RDS allocated storage (GB)"
  type        = number
  default     = 100
}

variable "db_username" {
  description = "Database username"
  type        = string
  sensitive   = true
}

variable "db_password" {
  description = "Database password"
  type        = string
  sensitive   = true
}

variable "redis_node_type" {
  description = "Redis node type"
  type        = string
  default     = "cache.t3.medium"
}

# Data sources
data "aws_availability_zones" "available" {
  state = "available"
}

data "aws_caller_identity" "current" {}

# Outputs
output "eks_cluster_endpoint" {
  value = aws_eks_cluster.ebot.endpoint
}

output "eks_cluster_name" {
  value = aws_eks_cluster.ebot.name
}

output "rds_endpoint" {
  value     = aws_db_instance.ebot.endpoint
  sensitive = true
}

output "redis_endpoint" {
  value = aws_elasticache_cluster.ebot.cache_nodes[0].address
}

output "s3_bucket_name" {
  value = aws_s3_bucket.ebot.id
}
