import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:hamsa/core/l10n/app_localizations.dart';
import 'package:persian_datetime_picker/persian_datetime_picker.dart';

import 'core/router/app_router.dart';
import 'core/theme/app_theme.dart';

class MainApp extends ConsumerWidget {
  const MainApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return MaterialApp.router(
      onGenerateTitle: (context) => AppLocalizations.of(context).appTitle,
      routerConfig: ref.watch(routerProvider),
      theme: AppTheme.light,

      // Persian-only product (plan.md): single `fa` locale, RTL follows from
      // the locale, no English support in P0.
      // fa_IR (not bare fa): persian_datetime_picker's
      // PersianMaterialLocalizations only activates for countryCode == 'IR',
      // and without it the picker dialog renders Gregorian month names
      // (e.g. "اوت") over Jalali years — a mixed calendar.
      locale: const Locale('fa', 'IR'),
      supportedLocales: const [Locale('fa', 'IR')],
      localizationsDelegates: const [
        AppLocalizations.delegate,
        PersianMaterialLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
    );
  }
}

void main() {
  runApp(const ProviderScope(child: MainApp()));
}
