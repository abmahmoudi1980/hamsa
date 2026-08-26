import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:hamsa/core/datetime/jalali.dart';
import 'package:hamsa/core/l10n/app_localizations.dart';
import 'package:hamsa/core/network/api_exception.dart';

import '../auth_controller.dart';
import '../auth_repository.dart';

/// US1 step 2 — OTP entry (T026): verify code, 60 s resend countdown, Persian
/// error states. On success the session is persisted and the router's
/// auth-state redirect lands the user on their role shell (manager →
/// building list, resident → resident panel).
class OtpScreen extends ConsumerStatefulWidget {
  const OtpScreen({super.key, required this.phone});

  final String phone;

  @override
  ConsumerState<OtpScreen> createState() => _OtpScreenState();
}

class _OtpScreenState extends ConsumerState<OtpScreen> {
  static const _resendWindow = Duration(seconds: 60);

  final _codeController = TextEditingController();
  final _formKey = GlobalKey<FormState>();
  Timer? _ticker;
  Duration _elapsed = Duration.zero;
  bool _submitting = false;

  @override
  void initState() {
    super.initState();
    // Dev convenience: server may return the code in the response.
    WidgetsBinding.instance.addPostFrameCallback((_) => _requestCode());
  }

  @override
  void dispose() {
    _ticker?.cancel();
    _codeController.dispose();
    super.dispose();
  }

  bool get _canResend => _elapsed >= _resendWindow;

  void _startCountdown() {
    _elapsed = Duration.zero;
    _ticker?.cancel();
    _ticker = Timer.periodic(const Duration(seconds: 1), (_) {
      if (!mounted) return;
      setState(() => _elapsed += const Duration(seconds: 1));
    });
  }

  Future<void> _requestCode({bool announce = false}) async {
    final messenger = ScaffoldMessenger.of(context);
    final l10n = AppLocalizations.of(context);
    _startCountdown();
    try {
      final devCode = await ref
          .read(authRepositoryProvider)
          .requestOtp(widget.phone);
      if (!mounted) return;
      if (announce && devCode != null && devCode.isNotEmpty) {
        messenger.showSnackBar(
          SnackBar(content: Text(toPersianDigits(devCode))),
        );
      }
    } on ApiException catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(l10n.apiErrorMessage(e))));
    }
  }

  Future<void> _verify() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() => _submitting = true);

    final messenger = ScaffoldMessenger.of(context);
    final l10n = AppLocalizations.of(context);

    try {
      final result = await ref
          .read(authRepositoryProvider)
          .verifyOtp(
            phone: widget.phone,
            code: fromPersianDigits(_codeController.text.trim()),
          );
      await ref
          .read(authControllerProvider.notifier)
          .completeLogin(
            user: result.user,
            accessToken: result.accessToken,
            refreshToken: result.refreshToken,
          );
      // Router redirect navigates to /manager or /home by role — no push here.
    } on ApiException catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(l10n.apiErrorMessage(e))));
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final remaining = _canResend ? null : _resendWindow - _elapsed;

    return Scaffold(
      appBar: AppBar(title: Text(l10n.verifyAndLogin)),
      body: SafeArea(
        child: Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Form(
              key: _formKey,
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    l10n.otpSentTo(toPersianDigits(widget.phone)),
                    textAlign: TextAlign.center,
                    style: Theme.of(context).textTheme.bodyMedium,
                  ),
                  const SizedBox(height: 24),
                  TextFormField(
                    controller: _codeController,
                    decoration: InputDecoration(labelText: l10n.otpLabel),
                    keyboardType: TextInputType.number,
                    textAlign: TextAlign.center,
                    autofocus: true,
                    maxLength: 6,
                    validator: (v) => (v == null || v.trim().isEmpty)
                        ? l10n.invalidOtp
                        : null,
                    onFieldSubmitted: (_) => _verify(),
                  ),
                  const SizedBox(height: 16),
                  FilledButton(
                    onPressed: _submitting ? null : _verify,
                    child: _submitting
                        ? const SizedBox(
                            width: 20,
                            height: 20,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : Text(l10n.verifyAndLogin),
                  ),
                  TextButton(
                    onPressed: _canResend
                        ? () => _requestCode(announce: true)
                        : null,
                    child: Text(
                      remaining == null
                          ? l10n.resendOtp
                          : l10n.resendIn(
                              toPersianDigits('${remaining.inSeconds}'),
                            ),
                    ),
                  ),
                  TextButton(
                    onPressed: () => context.pop(),
                    child: Text(l10n.changePhone),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
