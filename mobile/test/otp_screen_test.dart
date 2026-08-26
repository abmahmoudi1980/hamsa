import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:hamsa/core/l10n/app_localizations.dart';
import 'package:hamsa/core/network/api_exception.dart';
import 'package:hamsa/core/datetime/jalali.dart';
import 'package:hamsa/features/auth/auth_repository.dart';
import 'package:hamsa/features/auth/models/user_session.dart';
import 'package:hamsa/features/auth/screens/otp_screen.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter/material.dart';

class _StubbedRepository extends AuthRepository {
  _StubbedRepository() : super(Dio());

  int requestCalls = 0;
  int verifyCalls = 0;

  @override
  Future<String?> requestOtp(String phone) async {
    requestCalls++;
    return '482913'; // dev-mode code
  }

  @override
  Future<VerifyResult> verifyOtp({
    required String phone,
    required String code,
  }) async {
    verifyCalls++;
    if (code != fromPersianDigits('482913')) {
      throw ApiException(code: 'UNAUTHENTICATED', statusCode: 401);
    }
    return const VerifyResult(
      user: UserSession(id: 'u1', name: '', role: UserRole.manager),
      accessToken: 'a',
      refreshToken: 'r',
    );
  }
}

void main() {
  testWidgets('OTP screen shows target phone, verifies code, gates resend', (
    tester,
  ) async {
    final repo = _StubbedRepository();

    await tester.pumpWidget(
      ProviderScope(
        overrides: [authRepositoryProvider.overrideWithValue(repo)],
        child: const MaterialApp(
          locale: Locale('fa'),
          supportedLocales: AppLocalizations.supportedLocales,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          home: OtpScreen(phone: '09123456789'),
        ),
      ),
    );
    await tester.pump(); // post-frame callback fires requestOtp

    // Persian copy from fa.arb, phone rendered with Persian digits.
    expect(
      find.text('کد تأیید به شماره ۰۹۱۲۳۴۵۶۷۸۹ پیامک شد.'),
      findsOneWidget,
    );

    // Resend is throttled right after entry (60 s window).
    expect(find.text('ارسال مجدد کد تا ۶۰ ثانیه دیگر'), findsOneWidget);

    // Entering the dev code (Persian digits accepted) verifies and logs in.
    await tester.enterText(find.byType(TextFormField).first, '۴۸۲۹۱۳');
    await tester.tap(find.widgetWithText(FilledButton, 'تأیید و ورود'));
    await tester.pump();

    expect(repo.verifyCalls, 1);
    expect(repo.requestCalls, 1); // initial auto-request only
  });
}
