import 'package:flutter/material.dart';

import '../../core/theme/app_theme.dart';

/// Bottom-pinned primary action bar for form screens — set as the Scaffold's
/// [Scaffold.bottomNavigationBar]. Applies SafeArea plus a real bottom inset
/// ([AppTheme.spaceL]) so the button never sits flush against the screen
/// edge, and stretches the action to full width.
class SaveBar extends StatelessWidget {
  const SaveBar({super.key, required this.child});

  /// Typically a single full-width `FilledButton`.
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(
          AppTheme.spaceXl,
          AppTheme.spaceS,
          AppTheme.spaceXl,
          AppTheme.spaceL,
        ),
        child: SizedBox(width: double.infinity, child: child),
      ),
    );
  }
}
