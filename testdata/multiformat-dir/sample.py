import os
from dataclasses import dataclass
from typing import Optional

@dataclass
class Config:
    host: str
    port: int = 8080
    debug: bool = False

class UserRepository:
    def __init__(self, db):
        self.db = db

    def find_by_id(self, user_id: str) -> Optional[dict]:
        return self.db.query("SELECT * FROM users WHERE id = ?", user_id)

    def delete(self, user_id: str) -> bool:
        return self.db.execute("DELETE FROM users WHERE id = ?", user_id)

def create_app(config: Config):
    app = Flask(__name__)
    return app

async def fetch_data(url: str) -> bytes:
    async with aiohttp.ClientSession() as session:
        async with session.get(url) as resp:
            return await resp.read()
