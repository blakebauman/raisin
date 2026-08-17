"""Raisin Python SDK."""

from __future__ import annotations

import hashlib
import hmac
import json
import time
import urllib.error
import urllib.request
from typing import Any, Optional

DEFAULT_BASE_URL = "https://api.raisin.run"


class RaisinError(Exception):
    def __init__(self, name: str, message: str, status_code: int):
        super().__init__(message)
        self.name = name
        self.message = message
        self.status_code = status_code


class Raisin:
    def __init__(self, api_key: str, base_url: str = DEFAULT_BASE_URL):
        self.api_key = api_key
        self.base_url = base_url.rstrip("/")
        self.emails = Emails(self)
        self.domains = Domains(self)
        self.webhooks = Webhooks(self)
        self.api_keys = APIKeys(self)
        self.contacts = Contacts(self)
        self.templates = Templates(self)
        self.broadcasts = Broadcasts(self)

    def request(self, method: str, path: str, body: Any = None) -> Any:
        data = None if body is None else json.dumps(body).encode("utf-8")
        req = urllib.request.Request(
            self.base_url + path,
            data=data,
            method=method,
            headers={
                "Authorization": f"Bearer {self.api_key}",
                "Content-Type": "application/json",
                "User-Agent": "raisin-python/0.1.0",
            },
        )
        try:
            with urllib.request.urlopen(req, timeout=30) as res:
                return json.loads(res.read().decode("utf-8"))
        except urllib.error.HTTPError as e:
            raw = e.read().decode("utf-8")
            try:
                payload = json.loads(raw)
            except Exception:
                payload = {"name": "error", "message": raw, "statusCode": e.code}
            raise RaisinError(
                payload.get("name", "error"),
                payload.get("message", raw),
                payload.get("statusCode", e.code),
            ) from e


class Emails:
    def __init__(self, client: Raisin):
        self._c = client

    def send(
        self,
        *,
        from_: str,
        to: list[str] | str,
        subject: str,
        html: Optional[str] = None,
        text: Optional[str] = None,
        attachments: Optional[list[dict]] = None,
    ) -> dict:
        if isinstance(to, str):
            to = [to]
        body: dict = {"from": from_, "to": to, "subject": subject, "html": html, "text": text}
        if attachments:
            body["attachments"] = attachments
        return self._c.request("POST", "/emails", body)

    def get(self, id: str) -> dict:
        return self._c.request("GET", f"/emails/{id}")

    def list(self) -> dict:
        return self._c.request("GET", "/emails")

    def cancel(self, id: str) -> dict:
        return self._c.request("POST", f"/emails/{id}/cancel", {})

    def attachments(self, email_id: str) -> dict:
        return self._c.request("GET", f"/emails/{email_id}/attachments")


class Domains:
    def __init__(self, client: Raisin):
        self._c = client

    def create(self, name: str, region: str = "us-east-1") -> dict:
        return self._c.request("POST", "/domains", {"name": name, "region": region})

    def list(self) -> dict:
        return self._c.request("GET", "/domains")

    def get(self, id: str) -> dict:
        return self._c.request("GET", f"/domains/{id}")

    def verify(self, id: str) -> dict:
        return self._c.request("POST", f"/domains/{id}/verify", {})

    def remove(self, id: str) -> dict:
        return self._c.request("DELETE", f"/domains/{id}")


class Webhooks:
    def __init__(self, client: Raisin):
        self._c = client

    def create(self, endpoint: str, events: list[str]) -> dict:
        return self._c.request("POST", "/webhooks", {"endpoint": endpoint, "events": events})

    def list(self) -> dict:
        return self._c.request("GET", "/webhooks")

    def list_events(self, id: str) -> dict:
        return self._c.request("GET", f"/webhooks/{id}/events")

    def list_attempts(self, webhook_id: str, event_id: str) -> dict:
        return self._c.request("GET", f"/webhooks/{webhook_id}/events/{event_id}/attempts")

    def remove(self, id: str) -> dict:
        return self._c.request("DELETE", f"/webhooks/{id}")


class APIKeys:
    def __init__(self, client: Raisin):
        self._c = client

    def create(self, name: str) -> dict:
        return self._c.request("POST", "/api-keys", {"name": name})

    def list(self) -> dict:
        return self._c.request("GET", "/api-keys")


class Contacts:
    def __init__(self, client: Raisin):
        self._c = client

    def create(self, email: str) -> dict:
        return self._c.request("POST", "/contacts", {"email": email})

    def list(self) -> dict:
        return self._c.request("GET", "/contacts")


class Templates:
    def __init__(self, client: Raisin):
        self._c = client

    def create(self, **body: Any) -> dict:
        return self._c.request("POST", "/templates", body)

    def list(self) -> dict:
        return self._c.request("GET", "/templates")


class Broadcasts:
    def __init__(self, client: Raisin):
        self._c = client

    def create(self, **body: Any) -> dict:
        return self._c.request("POST", "/broadcasts", body)

    def send(self, id: str) -> dict:
        return self._c.request("POST", f"/broadcasts/{id}/send", {})


def verify_webhook_signature(
    secret: str,
    signature_header: str,
    body: bytes | str,
    *,
    max_skew_seconds: int = 300,
) -> bool:
    """Verify Raisin-Signature: t=<unix>,v1=<hex>."""
    parts: dict[str, str] = {}
    for p in signature_header.split(","):
        kv = p.strip().split("=", 1)
        if len(kv) == 2:
            parts[kv[0]] = kv[1]
    ts, sig = parts.get("t"), parts.get("v1")
    if not ts or not sig:
        return False
    try:
        sec = int(ts)
    except ValueError:
        return False
    if abs(int(time.time()) - sec) > max_skew_seconds:
        return False
    raw = body.encode("utf-8") if isinstance(body, str) else body
    expected = hmac.new(secret.encode("utf-8"), f"{ts}.".encode("utf-8") + raw, hashlib.sha256).hexdigest()
    return hmac.compare_digest(expected, sig)
