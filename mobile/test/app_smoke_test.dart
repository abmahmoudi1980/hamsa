import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:hamsa/core/l10n/app_localizations.dart';
import 'package:hamsa/main.dart';

void main() {
  testWidgets('app boots to login with fa locale and RTL layout', (
    tester,
  ) async {
    await tester.pumpWidget(const ProviderScope(child: MainApp()));

    // Session restore runs a microtask; let the router settle to /login.
    await tester.pumpAndSettle();

    final context = tester.element(find.byType(Scaffold).first);
    final l10n = AppLocalizations.of(context);

    expect(l10n.localeName, 'fa');
    expect(find.text(l10n.loginTitle), findsOneWidget);

    final directionality = tester.widget<Directionality>(
      find.ancestor(
        of: find.text(l10n.loginTitle),
        matching: find.byType(Directionality),
      ).first,
    );
    expect(directionality.textDirection, TextDirection.rtl);
  });
}
