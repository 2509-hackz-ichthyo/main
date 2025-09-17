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

# DynamoDB table for Game Service (Single Table Design)
resource "aws_dynamodb_table" "game_service" {
  name           = "game-service"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "PK"
  range_key      = "SK"

  attribute {
    name = "PK"
    type = "S"
  }

  attribute {
    name = "SK"
    type = "S"
  }

  attribute {
    name = "GSI1PK"
    type = "S"
  }

  attribute {
    name = "GSI1SK"
    type = "S"
  }

  global_secondary_index {
    name            = "GSI1"
    hash_key        = "GSI1PK"
    range_key       = "GSI1SK"
    projection_type = "ALL"
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
          "dynamodb:PutItem",
          "dynamodb:GetItem",
          "dynamodb:DeleteItem",
          "dynamodb:Query",
          "dynamodb:Scan",
          "dynamodb:UpdateItem"
        ]
        Resource = [
          aws_dynamodb_table.game_service.arn,
          "${aws_dynamodb_table.game_service.arn}/*"
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "dynamodb:PutItem",
          "dynamodb:GetItem"
        ]
        Resource = [
          aws_dynamodb_table.game_archive.arn,
          "${aws_dynamodb_table.game_archive.arn}/*"
        ]
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

# Data source to create ZIP file for matchmaking Lambda
data "archive_file" "lambda_matchmaking_zip" {
  type        = "zip"
  source_dir  = "${path.module}/lambda/matchmaking_handler"
  output_path = "${path.module}/lambda/matchmaking_handler.zip"
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

# Lambda function for WebSocket matchmaking route
resource "aws_lambda_function" "hackz_ichthyo_matchmaking_handler" {
  filename         = data.archive_file.lambda_matchmaking_zip.output_path
  function_name    = "hackz-ichthyo-websocket-matchmaking"
  role            = aws_iam_role.lambda_websocket_role.arn
  handler         = "bootstrap"
  runtime         = "provided.al2"
  timeout         = 30

  source_code_hash = data.archive_file.lambda_matchmaking_zip.output_base64sha256

  environment {
    variables = {
      DYNAMODB_TABLE_NAME = aws_dynamodb_table.game_service.name
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

# matchmaking lambda integration
resource "aws_apigatewayv2_integration" "hackz_ichthyo_matchmaking_integration" {
  api_id             = aws_apigatewayv2_api.hackz_ichthyo_websocket.id
  integration_type   = "AWS_PROXY"
  integration_method = "POST"
  integration_uri    = aws_lambda_function.hackz_ichthyo_matchmaking_handler.invoke_arn
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

# Data source to create ZIP file for disconnect Lambda
data "archive_file" "lambda_disconnect_zip" {
  type        = "zip"
  source_dir  = "${path.module}/lambda/disconnect_handler"
  output_path = "${path.module}/lambda/disconnect_handler.zip"
}

# Lambda function for WebSocket $disconnect route
resource "aws_lambda_function" "hackz_ichthyo_disconnect_handler" {
  filename         = data.archive_file.lambda_disconnect_zip.output_path
  function_name    = "hackz-ichthyo-websocket-disconnect"
  role            = aws_iam_role.lambda_websocket_role.arn
  handler         = "bootstrap"
  runtime         = "provided.al2"
  timeout         = 30

  source_code_hash = data.archive_file.lambda_disconnect_zip.output_base64sha256

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

# Output WebSocket URL (カスタムドメイン)
output "websocket_url" {
  description = "WebSocket API URL (カスタムドメイン)"
  value       = "wss://${aws_apigatewayv2_domain_name.websocket_custom_domain.domain_name}/${aws_apigatewayv2_stage.hackz_ichthyo_stage.name}"
}

# Output WebSocket URL (デフォルト)
output "websocket_url_default" {
  description = "WebSocket API URL (デフォルト)"
  value       = "${aws_apigatewayv2_api.hackz_ichthyo_websocket.api_endpoint}/production"
}

# Lambda integration for $disconnect
resource "aws_apigatewayv2_integration" "hackz_ichthyo_disconnect_integration" {
  api_id             = aws_apigatewayv2_api.hackz_ichthyo_websocket.id
  integration_type   = "AWS_PROXY"
  integration_method = "POST"
  integration_uri    = aws_lambda_function.hackz_ichthyo_disconnect_handler.invoke_arn
}

# $disconnect route
resource "aws_apigatewayv2_route" "hackz_ichthyo_disconnect" {
  api_id    = aws_apigatewayv2_api.hackz_ichthyo_websocket.id
  route_key = "$disconnect"
  target    = "integrations/${aws_apigatewayv2_integration.hackz_ichthyo_disconnect_integration.id}"
}

# joinGame route
resource "aws_apigatewayv2_route" "hackz_ichthyo_joingame" {
  api_id    = aws_apigatewayv2_api.hackz_ichthyo_websocket.id
  route_key = "joinGame"
  target    = "integrations/${aws_apigatewayv2_integration.hackz_ichthyo_matchmaking_integration.id}"
}

# permission for exec lambda (connect)
resource "aws_lambda_permission" "connect_handler" {
  statement_id  = "AllowExecutionFromAPIGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.hackz_ichthyo_connect_handler.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "arn:aws:execute-api:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:${aws_apigatewayv2_api.hackz_ichthyo_websocket.id}/*/$connect"
}

# permission for exec lambda (disconnect)
resource "aws_lambda_permission" "disconnect_handler" {
  statement_id  = "AllowDisconnectExecutionFromAPIGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.hackz_ichthyo_disconnect_handler.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "arn:aws:execute-api:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:${aws_apigatewayv2_api.hackz_ichthyo_websocket.id}/*/$disconnect"
}

# permission for exec lambda (matchmaking)
resource "aws_lambda_permission" "matchmaking_handler" {
  statement_id  = "AllowMatchmakingExecutionFromAPIGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.hackz_ichthyo_matchmaking_handler.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "arn:aws:execute-api:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:${aws_apigatewayv2_api.hackz_ichthyo_websocket.id}/*/joinGame"
}

# Data source to create ZIP file for game Lambda
data "archive_file" "lambda_game_zip" {
  type        = "zip"
  source_dir  = "${path.module}/lambda/game_handler"
  output_path = "${path.module}/lambda/game_handler.zip"
}

# Lambda function for WebSocket game route
resource "aws_lambda_function" "hackz_ichthyo_game_handler" {
  filename         = data.archive_file.lambda_game_zip.output_path
  function_name    = "hackz-ichthyo-websocket-game"
  role            = aws_iam_role.lambda_websocket_role.arn
  handler         = "bootstrap"
  runtime         = "provided.al2"
  timeout         = 30

  source_code_hash = data.archive_file.lambda_game_zip.output_base64sha256

  environment {
    variables = {
      DYNAMODB_TABLE_NAME = aws_dynamodb_table.game_service.name
    }
  }

  tags = {
    Environment = "hackathon"
    Project     = "ichthyo-reversi"
  }
}

# Lambda integration for game
resource "aws_apigatewayv2_integration" "hackz_ichthyo_game_integration" {
  api_id             = aws_apigatewayv2_api.hackz_ichthyo_websocket.id
  integration_type   = "AWS_PROXY"
  integration_method = "POST"
  integration_uri    = aws_lambda_function.hackz_ichthyo_game_handler.invoke_arn
}

# makeMove route
resource "aws_apigatewayv2_route" "hackz_ichthyo_makemove" {
  api_id    = aws_apigatewayv2_api.hackz_ichthyo_websocket.id
  route_key = "makeMove"
  target    = "integrations/${aws_apigatewayv2_integration.hackz_ichthyo_game_integration.id}"
}

# gameFinished route
resource "aws_apigatewayv2_route" "hackz_ichthyo_gamefinished" {
  api_id    = aws_apigatewayv2_api.hackz_ichthyo_websocket.id
  route_key = "gameFinished"
  target    = "integrations/${aws_apigatewayv2_integration.hackz_ichthyo_game_integration.id}"
}

# permission for exec lambda (game - makeMove)
resource "aws_lambda_permission" "game_handler_makemove" {
  statement_id  = "AllowGameExecutionFromAPIGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.hackz_ichthyo_game_handler.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "arn:aws:execute-api:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:${aws_apigatewayv2_api.hackz_ichthyo_websocket.id}/*/makeMove"
}

# permission for exec lambda (game - gameFinished)
resource "aws_lambda_permission" "game_handler_gamefinished" {
  statement_id  = "AllowGameFinishExecutionFromAPIGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.hackz_ichthyo_game_handler.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "arn:aws:execute-api:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:${aws_apigatewayv2_api.hackz_ichthyo_websocket.id}/*/gameFinished"
}

# custom domain
# certificate
resource "aws_acm_certificate" "websocket_cert" {
  domain_name       = "2509-hackz-ichthyo.ulxsth.com"
  validation_method = "DNS"
  
  lifecycle {
    create_before_destroy = true
  }
  
  tags = {
    Environment = "hackathon"
    Project     = "ichthyo-reversi"
    Domain      = "2509-hackz-ichthyo.ulxsth.com"
  }
}

# ===== CLEANUP HANDLER =====

# Data source to create ZIP file for Cleanup Lambda
data "archive_file" "cleanup_handler_zip" {
  type        = "zip"
  source_dir  = "${path.module}/lambda/cleanup_handler"
  output_path = "${path.module}/lambda/cleanup_handler.zip"
  excludes    = ["*.zip", "go.sum"]
}

# Cleanup Handler Lambda Function
resource "aws_lambda_function" "cleanup_handler" {
  filename         = data.archive_file.cleanup_handler_zip.output_path
  function_name    = "hackz-ichthyo-cleanup-handler"
  role            = aws_iam_role.lambda_cleanup_role.arn
  handler         = "bootstrap"
  source_code_hash = data.archive_file.cleanup_handler_zip.output_base64sha256
  runtime         = "provided.al2"
  timeout         = 300 # 5 minutes

  environment {
    variables = {
      GAME_SERVICE_TABLE_NAME   = aws_dynamodb_table.game_service.name
      WEBSOCKET_TABLE_NAME      = aws_dynamodb_table.websocket_connections.name
    }
  }

  tags = {
    Environment = "hackathon"
    Project     = "ichthyo-reversi"
  }

  depends_on = [
    aws_iam_role_policy.lambda_cleanup_policy,
    aws_cloudwatch_log_group.cleanup_handler_logs,
  ]
}

# CloudWatch Log Group for Cleanup Handler
resource "aws_cloudwatch_log_group" "cleanup_handler_logs" {
  name              = "/aws/lambda/hackz-ichthyo-cleanup-handler"
  retention_in_days = 7

  tags = {
    Environment = "hackathon"
    Project     = "ichthyo-reversi"
  }
}

# IAM Role for Cleanup Lambda
resource "aws_iam_role" "lambda_cleanup_role" {
  name = "hackz-ichthyo-cleanup-lambda-role"

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

# IAM Policy for Cleanup Lambda
resource "aws_iam_role_policy" "lambda_cleanup_policy" {
  name = "hackz-ichthyo-cleanup-lambda-policy"
  role = aws_iam_role.lambda_cleanup_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:Query",
          "dynamodb:Scan",
          "dynamodb:DeleteItem",
          "dynamodb:BatchWriteItem"
        ]
        Resource = [
          aws_dynamodb_table.game_service.arn,
          "${aws_dynamodb_table.game_service.arn}/*",
          aws_dynamodb_table.websocket_connections.arn
        ]
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

# EventBridge Rule for periodic cleanup (every 5 minutes)
resource "aws_cloudwatch_event_rule" "cleanup_schedule" {
  name                = "hackz-ichthyo-cleanup-schedule"
  description         = "Trigger cleanup handler every 5 minutes"
  schedule_expression = "rate(5 minutes)"

  tags = {
    Environment = "hackathon"
    Project     = "ichthyo-reversi"
  }
}

# EventBridge Target to invoke Cleanup Lambda
resource "aws_cloudwatch_event_target" "cleanup_target" {
  rule      = aws_cloudwatch_event_rule.cleanup_schedule.name
  target_id = "CleanupHandlerTarget"
  arn       = aws_lambda_function.cleanup_handler.arn
}

# Permission for EventBridge to invoke Cleanup Lambda
resource "aws_lambda_permission" "allow_eventbridge_cleanup" {
  statement_id  = "AllowExecutionFromEventBridge"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.cleanup_handler.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.cleanup_schedule.arn
}

# 証明書検証の完了確認
resource "aws_acm_certificate_validation" "websocket_cert_validation" {
  certificate_arn = aws_acm_certificate.websocket_cert.arn
  
  timeouts {
    create = "10m"  # 最大10分間検証を待機
  }
}

# API Gateway カスタムドメイン
resource "aws_apigatewayv2_domain_name" "websocket_custom_domain" {
  domain_name = "2509-hackz-ichthyo.ulxsth.com"

  domain_name_configuration {
    certificate_arn = aws_acm_certificate_validation.websocket_cert_validation.certificate_arn
    endpoint_type   = "REGIONAL"
    security_policy = "TLS_1_2"
  }
  
  tags = {
    Environment = "hackathon"
    Project     = "ichthyo-reversi"
  }
}

# API マッピング
resource "aws_apigatewayv2_api_mapping" "websocket_mapping" {
  api_id      = aws_apigatewayv2_api.hackz_ichthyo_websocket.id
  domain_name = aws_apigatewayv2_domain_name.websocket_custom_domain.id
  stage       = aws_apigatewayv2_stage.hackz_ichthyo_stage.id
}

# ACM証明書ARNの出力
output "acm_certificate_arn" {
  description = "ACM証明書のARN"
  value       = aws_acm_certificate.websocket_cert.arn
}

output "acm_dns_validation_info" {
  description = "ACM証明書のDNS検証レコード情報（Cloudflareに設定が必要）"
  value = {
    for dvo in aws_acm_certificate.websocket_cert.domain_validation_options : dvo.domain_name => {
      validation_name   = dvo.resource_record_name
      validation_value  = dvo.resource_record_value
      validation_type   = dvo.resource_record_type
    }
  }
}

# DynamoDB table for Game Archive
resource "aws_dynamodb_table" "game_archive" {
  name           = "game-archive"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "PK"
  range_key      = "SK"

  attribute {
    name = "PK"
    type = "S"
  }

  attribute {
    name = "SK"
    type = "S"
  }

  ttl {
    attribute_name = "ttl"
    enabled        = true
  }

  tags = {
    Environment = "hackathon"
    Project     = "ichthyo-reversi"
  }
}

# API Gateway ターゲットドメイン情報の出力
output "api_gateway_target_domain" {
  description = "CloudflareのCNAMEレコードに設定するターゲットドメイン"
  value = {
    target_domain_name = aws_apigatewayv2_domain_name.websocket_custom_domain.domain_name_configuration[0].target_domain_name
    hosted_zone_id     = aws_apigatewayv2_domain_name.websocket_custom_domain.domain_name_configuration[0].hosted_zone_id
  }
}