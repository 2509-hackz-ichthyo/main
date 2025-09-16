# https://zenn.dev/unilorn/articles/b4f64cc291cc82

# api gateway
resource "aws_apigatewayv2_api" "hackz_ichthyo_websocket" {
  name                       = "hackz_ichthyo-websocket"
  protocol_type              = "WEBSOCKET"
  route_selection_expression = "$request.body.action"
}

# endpoint ($connect)
resource "aws_apigatewayv2_route" "hackz_ichthyo_connect" {
  api_id    = aws_apigatewayv2_api.hackz_ichthyo_websocket.id
  route_key = "GET /ws"
  target    = "integrations/${aws_apigatewayv2_integration.hackz_ichthyo_integration.id}"
}

# TODO: ほかのもはやす

# lambda integration
# TODO: lambda は別で立てる（か追記する）
resource "aws_apigatewayv2_integration" "hackz_ichthyo_integration" {
  api_id             = aws_apigatewayv2_api.hackz_ichthyo_websocket.id
  integration_type   = "AWS_PROXY"
  integration_method = "POST"
  integration_uri    = aws_lambda_function.hackz_ichthyo_lambda.arn
}

# deployment stage
resource "aws_apigatewayv2_stage" "hackz_ichthyo_stage" {
  api_id      = aws_apigatewayv2_api.hackz_ichthyo_websocket.id
  name        = "$default"

  default_route_settings {
    data_trace_enabled = true
    detailed_metrics_enabled = true
    logging_level = "INFO"
    throttling_burst_limit = 5000
    throttling_rate_limit = 10000
  }

  deployment_id = aws_apigatewayv2_deployment.hackz_ichthyo_deployment.id
  depends_on = [ aws_apigatewayv2_deployment.hackz_ichthyo_deployment ]

  access_log_settings {
      destination_arn = aws_cloudwatch_log_group.hackz_ichthyo_log_group.arn
      format          = "$context.identity.sourceIp - - [$context.requestTime] \"$context.httpMethod $context.routeKey $context.protocol\" $context.status $context.responseLength $context.requestId $context.integrationErrorMessage"
  }
}

# deployment
resource "aws_apigatewayv2_deployment" "hackz_ichthyo_deployment" {
  api_id = aws_apigatewayv2_api.hackz_ichthyo_websocket.id

  triggers = {
    redeployment = sha1(join(",", tolist([
      jsonencode(aws_apigatewayv2_integration.hackz_ichthyo_integration),
      jsonencode(aws_apigatewayv2_route.hackz_ichthyo_connect),
    ])))
  }

  lifecycle {
    create_before_destroy = true
  }
}

# TODO: custom domain

# permission for exec lambda
resource "aws_lambda_permission" "connect_handler" {
  statement_id  = "AllowExecutionFromAPIGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.hackz_ichthyo_handler_connect.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "arn:aws:execute-api:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:${aws_apigatewayv2_api.hackz_ichthyo_websocket.id}/*/${aws_apigatewayv2_route.hackz_ichthyo_connect.route_key}"
}