import 'package:flutter/material.dart';
import 'package:hamsa/core/l10n/app_localizations.dart';

/// Resident navigation shell — full panel arrives in US9 (T083/T084).
class ResidentShellScreen extends StatelessWidget {
  const ResidentShellScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.residentShellTitle)),
      body: Center(child: Text(l10n.shellUnderConstruction)),
    );
  }
}
