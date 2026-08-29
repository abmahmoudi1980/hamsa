import '../../core/l10n/app_localizations.dart';

/// Charge-calculation method tokens (contracts/api.md: enum values stay
/// locale-neutral tokens; the client maps them to Persian labels).
const calcMethods = [
  'equal',
  'per_occupant',
  'per_area',
  'fixed',
  'specific_units',
  'combined',
];

/// Persian label for a calculation-method token. Unknown values fall back to
/// the combined label so new server-side tokens never render blank.
String calcMethodLabel(AppLocalizations l10n, String method) => switch (method) {
      'equal' => l10n.methodEqual,
      'per_occupant' => l10n.methodPerOccupant,
      'per_area' => l10n.methodPerArea,
      'fixed' => l10n.methodFixed,
      'specific_units' => l10n.methodSpecificUnits,
      _ => l10n.methodCombined,
    };
