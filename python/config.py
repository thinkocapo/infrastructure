import os
import socket
from dotenv import load_dotenv

load_dotenv()

SENTRY_DSN = os.environ["SENTRY_DSN"]
INTERVAL_SECONDS = int(os.getenv("INTERVAL_SECONDS", "60"))
HOST_TAG = socket.gethostname() or "unknown"
