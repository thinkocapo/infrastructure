import os
from dotenv import load_dotenv

load_dotenv()

SENTRY_DSN = os.environ["SENTRY_DSN"]
INTERVAL_SECONDS = int(os.getenv("INTERVAL_SECONDS", "60"))
