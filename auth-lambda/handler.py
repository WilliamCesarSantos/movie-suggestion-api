import json
import os
from jwt_service import JWTService

jwt_service = JWTService(
    secret=os.environ.get("JWT_SECRET", "dev-secret"),
    algorithm=os.environ.get("JWT_ALGORITHM", "HS256"),
    expiry_hours=int(os.environ.get("JWT_EXPIRY_HOURS", "24")),
)

def lambda_handler(event, context):
    action = event.get("action")
    
    if action == "generate":
        user_id = event.get("userId")
        email = event.get("email")
        role = event.get("role", "user")
        
        if not user_id or not email:
            return {"valid": False, "error": "userId and email are required"}
        
        try:
            token, expires_at = jwt_service.generate(user_id, email, role)
            return {
                "valid": True,
                "token": token,
                "userId": user_id,
                "email": email,
                "role": role,
                "expiresAt": expires_at,
            }
        except Exception as e:
            return {"valid": False, "error": str(e)}
    
    elif action == "validate":
        token = event.get("token")
        if not token:
            return {"valid": False, "error": "token is required"}
        
        try:
            claims = jwt_service.validate(token)
            return {
                "valid": True,
                "userId": claims.get("sub"),
                "email": claims.get("email"),
                "role": claims.get("role", "user"),
            }
        except Exception as e:
            return {"valid": False, "error": str(e)}
    
    else:
        return {"valid": False, "error": f"unknown action: {action}"}
