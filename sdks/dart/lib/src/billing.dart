import 'package:dio/dio.dart';

class Billing {
  final Dio _dio;

  Billing(this._dio);

  Future<Map<String, dynamic>> listPlans() async {
    final res = await _dio.get('/v1/billing/plans');
    return res.data;
  }

  Future<Map<String, dynamic>> getSubscription() async {
    final res = await _dio.get('/v1/billing/subscription');
    return res.data;
  }

  Future<Map<String, dynamic>> subscribe({
    required String planId,
    String? paymentMethodId,
  }) async {
    final res = await _dio.post('/v1/billing/subscription', data: {
      'planId': planId,
      if (paymentMethodId != null) 'paymentMethodId': paymentMethodId,
    });
    return res.data;
  }

  Future<void> cancelSubscription() async {
    await _dio.delete('/v1/billing/subscription');
  }

  Future<Map<String, dynamic>> getUsage() async {
    final res = await _dio.get('/v1/billing/usage');
    return res.data;
  }

  Future<Map<String, dynamic>> listInvoices() async {
    final res = await _dio.get('/v1/billing/invoices');
    return res.data;
  }

  Future<Map<String, dynamic>> getInvoice(String invoiceId) async {
    final res = await _dio.get('/v1/billing/invoices/$invoiceId');
    return res.data;
  }
}
