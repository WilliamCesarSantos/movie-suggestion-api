#!/bin/bash
awslocal ssm put-parameter --name /movie-suggestion/jwt-secret --value "dev-secret" --type SecureString --overwrite
