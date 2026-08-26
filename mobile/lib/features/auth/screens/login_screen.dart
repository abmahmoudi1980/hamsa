import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:hamsa/core/l10n/app_localizations.dart';
import 'package:hamsa/shared/validation/validators.dart';

import '../auth_controller.dart';
import '../models/user_session.dart';

/// Foundation shell for the login flow. The real phone-entry / OTP screens are
/// implemented in US1 (T026); this screen proves routing, locale and theming.
class LoginScreen extends ConsumerWidget {
  const LoginScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  l10n.loginTitle,
                  textAlign: TextAlign.center,
                  style: Theme.of(context).textTheme.headlineSmall,
                ),
                const SizedBox(height: 8),
                Text(
                  l10n.loginSubtitle,
                  textAlign: TextAlign.center,
                  style: Theme.of(context).textTheme.bodyMedium,
                ),
                const SizedBox(height: 24),
                TextFormField(
                  decoration: InputDecoration(labelText: l10n.phoneLabel),
                  keyboardType: TextInputType.phone,
                  autofocus: true,
                  validator: (v) => Validators.mobile(l10n, v),
                ),
                const SizedBox(height: 16),
                FilledButton(
                  onPressed: () => _devLogin(context, ref),
                  child: Text(l10n.requestOtp),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  /// Temporary dev bridge so the router's authenticated shells are reachable
  /// before the OTP flow lands; replaced by T026/T027 wiring.
  void _devLogin(BuildContext context, WidgetRef ref) {
    final messenger = ScaffoldMessenger.of(context);
    final router = GoRouter.of(context);
    ref
        .read(authControllerProvider.notifier)
        .completeLogin(
          user: const UserSession(id: 'dev', name: '', role: UserRole.manager),
          accessToken: 'dev-access',
          refreshToken: 'dev-refresh',
        )
        .then(
          (_) => router.go('/manager'),
          onError: (Object _) => messenger.showSnackBar(
            const SnackBar(content: Text('session store unavailable')),
          ),
        );
  }
}
