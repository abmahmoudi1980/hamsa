
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:hamsa/core/storage/token_storage.dart';
import 'package:hamsa/features/auth/auth_controller.dart';
import 'package:hamsa/features/auth/screens/manager_invite_screen.dart';
import 'package:hamsa/main.dart';

/// Serves a persisted superadmin session without touching platform channels.
class _SuperadminStorage extends TokenStorage {
  @override
  Future<StoredSession?> read() async => const StoredSession(
    accessToken: 'a',
    refreshToken: 'r',
    userJson: '{"id":"s1","name":"سرپرست ارشد","role":"superadmin"}',
  );
}

void main() {
  // 002-multi-manager-support T030/T031: the superadmin reaches the manager
  // shell; its home is only the invite tool (no building fetch, no building
  // dashboard).
  testWidgets('superadmin lands on the invite-only manager home', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [tokenStorageProvider.overrideWithValue(
          _SuperadminStorage(),
        )],
        child: const MainApp(),
      ),
    );
    await tester.pumpAndSettle();

    // Manager-shell home with the superadmin heading, not the resident shell
    // and not the building dashboard.
    expect(find.text('پنل سرپرست ارشد'), findsOneWidget);
    expect(find.text('داشبورد'), findsNothing);
    expect(find.text('ساختمان‌ها'), findsNothing);

    // The invite tool opens straight from the home CTA.
    await tester.tap(find.text('دریافت کد دعوت'));
    await tester.pumpAndSettle();
    expect(find.byType(ManagerInviteScreen), findsOneWidget);
  });
}
