import jwt
import datetime


class JWTService:
    def __init__(self, secret: str, algorithm: str = "HS256", expiry_hours: int = 24):
        self.secret = secret
        self.algorithm = algorithm
        self.expiry_hours = expiry_hours

    def generate(self, user_id: str, email: str, role: str):
        now = datetime.datetime.utcnow()
        expires_at = now + datetime.timedelta(hours=self.expiry_hours)
        payload = {
            "sub": user_id,
            "email": email,
            "role": role,
            "iat": now,
            "exp": expires_at,
        }
        token = jwt.encode(payload, self.secret, algorithm=self.algorithm)
        return token, expires_at.isoformat() + "Z"

    def validate(self, token: str) -> dict:
        return jwt.decode(token, self.secret, algorithms=[self.algorithm])
