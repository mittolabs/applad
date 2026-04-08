import '../server_client.dart';

class Billing {
  final ApplAdServer _client;

  Billing(this._client);

  Future<Map<String, dynamic>> listPlans() async {
    return _client.get('/v1/billing/plans');
  }

  Future<Map<String, dynamic>> getSubscription() async {
    return _client.get('/v1/billing/subscription');
  }

  Future<Map<String, dynamic>> subscribe({
    required String planId,
    String? paymentMethodId,
  }) async {
    return _client.post('/v1/billing/subscription', data: {
      'planId': planId,
      if (paymentMethodId != null) 'paymentMethodId': paymentMethodId,
    });
  }

  Future<Map<String, dynamic>> cancelSubscription() async {
    return _client.delete('/v1/billing/subscription');
  }

  Future<Map<String, dynamic>> getUsage() async {
    return _client.get('/v1/billing/usage');
  }

  Future<Map<String, dynamic>> listInvoices() async {
    return _client.get('/v1/billing/invoices');
  }

  Future<Map<String, dynamic>> getInvoice(String invoiceId) async {
    return _client.get('/v1/billing/invoices/$invoiceId');
  }
}
