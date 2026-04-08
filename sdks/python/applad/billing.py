"""Billing service."""


class Billing:
    def __init__(self, client):
        self.client = client

    def list_plans(self):
        return self.client._call("GET", "/billing/plans")

    def get_subscription(self):
        return self.client._call("GET", "/billing/subscription")

    def subscribe(self, plan_id: str, payment_method_id: str = None):
        data = {"planId": plan_id}
        if payment_method_id is not None:
            data["paymentMethodId"] = payment_method_id
        return self.client._call("POST", "/billing/subscription", data)

    def cancel_subscription(self):
        return self.client._call("DELETE", "/billing/subscription")

    def get_usage(self):
        return self.client._call("GET", "/billing/usage")

    def list_invoices(self):
        return self.client._call("GET", "/billing/invoices")

    def get_invoice(self, invoice_id: str):
        return self.client._call("GET", f"/billing/invoices/{invoice_id}")