# https://zenn.dev/unilorn/articles/b4f64cc291cc82

# DynamoDB table for WebSocket connections
resource "aws_dynamodb_table" "websocket_connections" {
  name           = "websocket-connections"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "connectionId"

  attribute {
    name = "connectionId"
    type = "S"
  }

  tags = {
    Environment = "hackathon"
    Project     = "ichthyo-reversi"
  }
}

# api gateway
resource "aws_apigatewayv2_api" "hackz_ichthyo_websocket" {
  name                       = "hackz_ichthyo-websocket"
  protocol_type              = "WEBSOCKET"
  route_selection_expression = "$request.body.action"
}

# endpoint ($connect)
resource "aws_apigatewayv2_route" "hackz_ichthyo_connect" {
  api_id    = aws_apigatewayv2_api.hackz_ichthyo_websocket.id
  route_key = "$connect"
  target    = "integrations/${aws_apigatewayv2_integration.hackz_ichthyo_integration.id}"
}

# IAM role for Lambda function
resource "aws_iam_role" "lambda_websocket_role" {
  name = "lambda-websocket-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Environment = "hackathon"
    Project     = "ichthyo-reversi"
  }
}

# IAM policy for Lambda to access DynamoDB and API Gateway Management API
resource "aws_iam_role_policy" "lambda_websocket_policy" {
  name = "lambda-websocket-policy"
  role = aws_iam_role.lambda_websocket_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:PutItem",
          "dynamodb:GetItem",
          "dynamodb:DeleteItem",
          "dynamodb:Scan"
        ]
        Resource = aws_dynamodb_table.websocket_connections.arn
      },
      {
        Effect = "Allow"
        Action = [
          "execute-api:ManageConnections"
        ]
        Resource = "${aws_apigatewayv2_api.hackz_ichthyo_websocket.execution_arn}/*/*"
      },
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "arn:aws:logs:*:*:*"
      }
    ]
  })
}

# Data source to create ZIP file for Lambda
data "archive_file" "lambda_zip" {
  type        = "zip"
  source_dir  = "${path.module}/lambda/connect_handler"
  output_path = "${path.module}/lambda/connect_handler.zip"
}

# Lambda function for WebSocket $connect route
resource "aws_lambda_function" "hackz_ichthyo_connect_handler" {
  filename         = data.archive_file.lambda_zip.output_path
  function_name    = "hackz-ichthyo-websocket-connect"
  role            = aws_iam_role.lambda_websocket_role.arn
  handler         = "bootstrap"
  runtime         = "provided.al2"
  timeout         = 30

  source_code_hash = data.archive_file.lambda_zip.output_base64sha256

  environment {
    variables = {
      DYNAMODB_TABLE_NAME = aws_dynamodb_table.websocket_connections.name
    }
  }

  tags = {
    Environment = "hackathon"
    Project     = "ichthyo-reversi"
  }
}

# lambda integration
resource "aws_apigatewayv2_integration" "hackz_ichthyo_integration" {
  api_id             = aws_apigatewayv2_api.hackz_ichthyo_websocket.id
  integration_type   = "AWS_PROXY"
  integration_method = "POST"
  integration_uri    = aws_lambda_function.hackz_ichthyo_connect_handler.invoke_arn
}

# deployment stage
resource "aws_apigatewayv2_stage" "hackz_ichthyo_stage" {
  api_id      = aws_apigatewayv2_api.hackz_ichthyo_websocket.id
  name        = "production"
  auto_deploy = true

  default_route_settings {
    data_trace_enabled = false
    detailed_metrics_enabled = false
    throttling_burst_limit = 5000
    throttling_rate_limit = 10000
  }

# Commented out access_log_settings to avoid CloudWatch Logs role requirement
  # access_log_settings {
  #     destination_arn = aws_cloudwatch_log_group.hackz_ichthyo_log_group.arn
  #     format          = "$context.identity.sourceIp - - [$context.requestTime] \"$context.httpMethod $context.routeKey $context.protocol\" $context.status $context.responseLength $context.requestId $context.integrationErrorMessage"
  # }

  tags = {
    Environment = "hackathon"
    Project     = "ichthyo-reversi"
  }
}

# Output WebSocket URL
output "websocket_url" {
  description = "WebSocket API URL"
  value       = "${aws_apigatewayv2_api.hackz_ichthyo_websocket.api_endpoint}/production"
}

# permission for exec lambda
resource "aws_lambda_permission" "connect_handler" {
  statement_id  = "AllowExecutionFromAPIGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.hackz_ichthyo_connect_handler.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "arn:aws:execute-api:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:${aws_apigatewayv2_api.hackz_ichthyo_websocket.id}/*/$connect"
}