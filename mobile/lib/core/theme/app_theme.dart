import 'package:flutter/material.dart';

/// App-wide design system: "calm fintech" — deep teal on warm neutrals,
/// generous rounding, tonal surfaces, Vazirmatn typography tuned for
/// Persian line-height. RTL handled by the `fa` locale at MaterialApp level.
abstract final class AppTheme {
  // ---- Design tokens (spacing scale) --------------------------------------
  static const double spaceXs = 4;
  static const double spaceS = 8;
  static const double spaceM = 12;
  static const double spaceL = 16;
  static const double spaceXl = 20;
  static const double spaceXxl = 24;
  static const double spaceXxxl = 32;

  /// Standard horizontal page padding.
  static const EdgeInsets pagePadding = EdgeInsets.symmetric(
    horizontal: spaceXl,
    vertical: spaceL,
  );

  /// Standard card/content corner radius.
  static const double radius = 16;
  static const double radiusLarge = 20;
  static const double radiusSmall = 12;

  /// Standard control height for full-width buttons.
  static const double buttonHeight = 52;

  // ---- Color ----------------------------------------------------------------
  /// Primary brand: deep teal. Surfaces warm slightly toward cream so the
  /// app doesn't read as a default Material template.
  static const Color _seed = Color(0xFF00695C);

  static ThemeData get light {
    final scheme = ColorScheme.fromSeed(
      seedColor: _seed,
      // Warm neutral background family instead of cold gray.
      surface: const Color(0xFFFAF9F6),
    );

    final textTheme = _buildTextTheme(scheme);

    return ThemeData(
      useMaterial3: true,
      colorScheme: scheme,
      fontFamily: 'Vazirmatn',
      scaffoldBackgroundColor: scheme.surface,
      textTheme: textTheme,
      splashFactory: InkSparkle.splashFactory,
      appBarTheme: AppBarTheme(
        centerTitle: true,
        backgroundColor: scheme.surface,
        foregroundColor: scheme.onSurface,
        elevation: 0,
        scrolledUnderElevation: 0.5,
        titleTextStyle: textTheme.titleLarge,
        shape: const Border(),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: scheme.surfaceContainerHighest.withValues(alpha: 0.5),
        contentPadding: const EdgeInsets.symmetric(
          horizontal: spaceL,
          vertical: 14,
        ),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(radiusSmall + 2),
          borderSide: BorderSide.none,
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(radiusSmall + 2),
          borderSide: BorderSide(color: scheme.outlineVariant, width: 1),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(radiusSmall + 2),
          borderSide: BorderSide(color: scheme.primary, width: 1.6),
        ),
        errorBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(radiusSmall + 2),
          borderSide: BorderSide(color: scheme.error, width: 1),
        ),
        focusedErrorBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(radiusSmall + 2),
          borderSide: BorderSide(color: scheme.error, width: 1.6),
        ),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          minimumSize: const Size.fromHeight(buttonHeight),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(radiusSmall + 2),
          ),
          textStyle: textTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.w700,
          ),
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          minimumSize: const Size.fromHeight(buttonHeight),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(radiusSmall + 2),
          ),
          side: BorderSide(color: scheme.outlineVariant),
          textStyle: textTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.w700,
          ),
        ),
      ),
      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(
          textStyle: textTheme.labelLarge?.copyWith(
            fontWeight: FontWeight.w700,
          ),
        ),
      ),
      cardTheme: CardThemeData(
        elevation: 0,
        color: scheme.surfaceContainerLow,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(radius),
        ),
        margin: EdgeInsets.zero,
        clipBehavior: Clip.antiAlias,
      ),
      chipTheme: ChipThemeData(
        backgroundColor: scheme.surfaceContainerHigh,
        side: BorderSide.none,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(999),
        ),
        labelStyle: textTheme.labelMedium,
        padding: const EdgeInsets.symmetric(horizontal: spaceM, vertical: 2),
      ),
      dividerTheme: DividerThemeData(
        color: scheme.outlineVariant.withValues(alpha: 0.5),
        thickness: 1,
        space: 1,
      ),
      listTileTheme: ListTileThemeData(
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(radius),
        ),
        contentPadding: const EdgeInsets.symmetric(
          horizontal: spaceL,
          vertical: 4,
        ),
        iconColor: scheme.primary,
      ),
      snackBarTheme: SnackBarThemeData(
        behavior: SnackBarBehavior.floating,
        backgroundColor: scheme.inverseSurface,
        contentTextStyle: textTheme.bodyMedium?.copyWith(
          color: scheme.onInverseSurface,
        ),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(radiusSmall),
        ),
      ),
      dialogTheme: DialogThemeData(
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(radiusLarge),
        ),
        titleTextStyle: textTheme.titleLarge,
        barrierColor: scheme.scrim.withValues(alpha: 0.4),
      ),
      bottomSheetTheme: const BottomSheetThemeData(
        showDragHandle: true,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.vertical(
            top: Radius.circular(radiusLarge),
          ),
        ),
      ),
      progressIndicatorTheme: ProgressIndicatorThemeData(
        color: scheme.primary,
        linearTrackColor: scheme.surfaceContainerHighest,
      ),
      floatingActionButtonTheme: FloatingActionButtonThemeData(
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(radiusSmall + 4),
        ),
      ),
      segmentedButtonTheme: SegmentedButtonThemeData(
        style: ButtonStyle(
          shape: WidgetStatePropertyAll(
            RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(radiusSmall),
            ),
          ),
          side: WidgetStatePropertyAll(
            BorderSide(color: scheme.outlineVariant),
          ),
        ),
      ),
    );
  }

  /// Vazirmatn text scale: Persian glyphs sit on a taller body, so body text
  /// gets line-height 1.7 and titles 1.3 to breathe. Weights lean on the
  /// medium/semibold side — Persian text reads thin at regular weights.
  static TextTheme _buildTextTheme(ColorScheme scheme) {
    const font = TextStyle(fontFamily: 'Vazirmatn', height: 1.7);
    const title = TextStyle(fontFamily: 'Vazirmatn', height: 1.3);
    return TextTheme(
      displaySmall: title.copyWith(
        fontSize: 34,
        fontWeight: FontWeight.w800,
        color: scheme.onSurface,
      ),
      headlineMedium: title.copyWith(
        fontSize: 28,
        fontWeight: FontWeight.w800,
        color: scheme.onSurface,
      ),
      headlineSmall: title.copyWith(
        fontSize: 24,
        fontWeight: FontWeight.w700,
        color: scheme.onSurface,
      ),
      titleLarge: title.copyWith(
        fontSize: 19,
        fontWeight: FontWeight.w700,
        color: scheme.onSurface,
      ),
      titleMedium: title.copyWith(
        fontSize: 16,
        fontWeight: FontWeight.w600,
        color: scheme.onSurface,
      ),
      titleSmall: title.copyWith(
        fontSize: 14,
        fontWeight: FontWeight.w600,
        color: scheme.onSurfaceVariant,
      ),
      bodyLarge: font.copyWith(fontSize: 16, color: scheme.onSurface),
      bodyMedium: font.copyWith(fontSize: 14, color: scheme.onSurface),
      bodySmall: font.copyWith(
        fontSize: 12.5,
        color: scheme.onSurfaceVariant,
        height: 1.6,
      ),
      labelLarge: title.copyWith(fontSize: 14, fontWeight: FontWeight.w700),
      labelMedium: title.copyWith(
        fontSize: 12.5,
        fontWeight: FontWeight.w600,
        color: scheme.onSurfaceVariant,
      ),
      labelSmall: title.copyWith(
        fontSize: 11,
        fontWeight: FontWeight.w600,
        color: scheme.onSurfaceVariant,
      ),
    );
  }
}
