import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:hamsa/core/l10n/app_localizations.dart';
import 'package:hamsa/core/network/api_exception.dart';

void main() {
  Future<String> describe(WidgetTester tester, Object error) async {
    String message = '';
    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('fa'),
        supportedLocales: AppLocalizations.supportedLocales,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        home: Builder(
          builder: (context) {
            message = describeError(AppLocalizations.of(context), error);
            return const SizedBox.shrink();
          },
        ),
      ),
    );
    return message;
  }

  testWidgets('connection failure carries network message + diagnostics', (
    tester,
  ) async {
    final dioError = DioException(
      type: DioExceptionType.connectionError,
      requestOptions: RequestOptions(
        baseUrl: 'http://192.168.1.123:8080/api/v1',
        path: '/auth/login',
      ),
    );

    final message = await describe(tester, dioError);

    expect(
      message,
      contains('خطا در اتصال به سرور'),
    );
    expect(message, contains('[diagnostic] connectionError'));
    expect(message, contains('192.168.1.123:8080'));
  });

  testWidgets('ApiException passes through as localized message', (
    tester,
  ) async {
    final message = await describe(
      tester,
      ApiException(code: 'RATE_LIMITED'),
    );
    expect(message, 'تعداد درخواست‌ها بیش از حد مجاز است؛ کمی صبر کنید.');
  });
}
