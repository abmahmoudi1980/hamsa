import 'package:flutter/material.dart';
import 'package:hamsa/core/l10n/app_localizations.dart';

/// Manager navigation shell — real dashboard arrives in US10 (T087).
class ManagerShellScreen extends StatelessWidget {
  const ManagerShellScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.managerShellTitle)),
      body: Center(child: Text(l10n.shellUnderConstruction)),
    );
  }
}
