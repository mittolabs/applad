"""Messaging service."""


class Messaging:
    def __init__(self, client):
        self.client = client

    def send_email(self, to: list, subject: str, html: str = ""):
        body = {"to": to, "subject": subject}
        if html:
            body["html"] = html
        return self.client._call("POST", "/messaging/email", body)

    def send_sms(self, to: list, content: str):
        return self.client._call("POST", "/messaging/sms", {
            "to": to,
            "content": content,
        })

    def send_push(self, to: list, title: str, body: str):
        return self.client._call("POST", "/messaging/push", {
            "to": to,
            "title": title,
            "body": body,
        })
