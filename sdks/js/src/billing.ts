import type { Applad } from './client';

export class Billing {
  constructor(private client: Applad) {}

  /** List available billing plans. */
  listPlans() {
    return this.client.call('GET', '/billing/plans');
  }

  /** Get the current subscription. */
  getSubscription() {
    return this.client.call('GET', '/billing/subscription');
  }

  /** Create or update a subscription. */
  subscribe(planId: string, opts?: { paymentMethodId?: string }) {
    return this.client.call('POST', '/billing/subscription', { planId, ...opts });
  }

  /** Cancel the current subscription. */
  cancelSubscription() {
    return this.client.call('DELETE', '/billing/subscription');
  }

  /** Get usage stats for the current billing period. */
  getUsage() {
    return this.client.call('GET', '/billing/usage');
  }

  /** List invoices. */
  listInvoices() {
    return this.client.call('GET', '/billing/invoices');
  }

  /** Get a specific invoice. */
  getInvoice(invoiceId: string) {
    return this.client.call('GET', `/billing/invoices/${invoiceId}`);
  }
}
