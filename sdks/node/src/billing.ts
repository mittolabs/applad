import type { ApplAdServer } from './client';

export class Billing {
  constructor(private client: ApplAdServer) {}

  listPlans() {
    return this.client.call('GET', '/billing/plans');
  }

  getSubscription() {
    return this.client.call('GET', '/billing/subscription');
  }

  subscribe(planId: string, opts?: { paymentMethodId?: string }) {
    return this.client.call('POST', '/billing/subscription', { planId, ...opts });
  }

  cancelSubscription() {
    return this.client.call('DELETE', '/billing/subscription');
  }

  getUsage() {
    return this.client.call('GET', '/billing/usage');
  }

  listInvoices() {
    return this.client.call('GET', '/billing/invoices');
  }

  getInvoice(invoiceId: string) {
    return this.client.call('GET', `/billing/invoices/${invoiceId}`);
  }
}
