import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:hamsa/core/l10n/app_localizations.dart';
import 'package:hamsa/features/auth/auth_repository.dart';
import 'package:hamsa/features/auth/screens/manager_invite_screen.dart';

import '../stub_http_adapter.dart';

/// Invite screen on a stubbed Dio: asserts the request payload carries the
/// role chosen on the ساکن/مدیر toggle and the banner names the issued role.
Future<void> _pumpInvite(WidgetTester tester, StubAdapter adapter) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        authRepositoryProvider.overrideWithValue(
          AuthRepository(stubDio(adapter)),
        ),
      ],
      child: const MaterialApp(
        locale: Locale('fa'),
        supportedLocales: AppLocalizations.supportedLocales,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        home: ManagerInviteScreen(),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  StubAdapter inviteAdapter({required String issuedRole}) => StubAdapter({
    'POST /auth/invites': (
      201,
      '{"code":"AB23CD45","role":"$issuedRole","expires_in_days":7}',
    ),
  });

  testWidgets('invite defaults to the ساکن role', (tester) async {
    await _pumpInvite(tester, inviteAdapter(issuedRole: 'resident'));

    final segments = tester.widget<SegmentedButton<String>>(
      find.byType(SegmentedButton<String>),
    );
    expect(segments.selected, {'resident'});
  });

  testWidgets('issuing sends the selected role and banners it', (
    tester,
  ) async {
    final adapter = inviteAdapter(issuedRole: 'manager');
    await _pumpInvite(tester, adapter);

    await tester.tap(find.text('مدیر'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextFormField), '09120000001');
    await tester.tap(find.text('دریافت کد دعوت'));
    await tester.pumpAndSettle();

    final invite = adapter.requests.single;
    expect(invite.method, 'POST');
    expect(invite.path, '/auth/invites');
    expect(invite.body, {'phone': '09120000001', 'role': 'manager'});

    // Banner states the issued role; code and expiry stay visible.
    expect(find.text('کد دعوت با نقش «مدیر» صادر شد.'), findsOneWidget);
    expect(find.text('AB23CD45'), findsOneWidget);
    expect(find.text('اعتبار کد: 7 روز'), findsOneWidget);
  });

  testWidgets('issuing without touching the toggle sends role=resident', (
    tester,
  ) async {
    final adapter = inviteAdapter(issuedRole: 'resident');
    await _pumpInvite(tester, adapter);

    await tester.enterText(find.byType(TextFormField), '09120000002');
    await tester.tap(find.text('دریافت کد دعوت'));
    await tester.pumpAndSettle();

    expect(adapter.requests.single.body, {
      'phone': '09120000002',
      'role': 'resident',
    });
    expect(find.text('کد دعوت با نقش «ساکن» صادر شد.'), findsOneWidget);
  });
}
