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

    # --- Templates ---

    def create_template(self, name: str, type_: str, subject: str, body: str,
                        variables: list | None = None, template_id: str = "unique()"):
        """Create a reusable message template with {{variable}} placeholders."""
        return self.client._call("POST", "/messaging/templates", {
            "templateId": template_id,
            "name": name,
            "type": type_,
            "subject": subject,
            "body": body,
            "variables": variables or [],
        })

    def list_templates(self):
        return self.client._call("GET", "/messaging/templates")

    def get_template(self, template_id: str):
        return self.client._call("GET", f"/messaging/templates/{template_id}")

    def update_template(self, template_id: str, **kwargs):
        return self.client._call("PUT", f"/messaging/templates/{template_id}", kwargs)

    def delete_template(self, template_id: str):
        return self.client._call("DELETE", f"/messaging/templates/{template_id}")

    def send_template(self, template_id: str, to: list,
                      variables: dict | None = None):
        """Render the template with variables and send to the given recipients."""
        return self.client._call(
            "POST",
            f"/messaging/templates/{template_id}/send",
            {"to": to, "variables": variables or {}},
        )
