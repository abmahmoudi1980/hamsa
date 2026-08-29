import 'package:flutter/material.dart';
import 'package:hamsa/core/l10n/app_localizations.dart';
import 'package:persian_datetime_picker/persian_datetime_picker.dart';

/// Jalali ↔ Gregorian conversion and Persian-digit formatting.
///
/// Storage and API stay Gregorian ISO-8601 (research.md R6); everything here
/// is presentation-only. [JalaliDatePickerField] below is the ONLY date input
/// component allowed anywhere in the app.

const List<String> _jalaliMonthNames = [
  'فروردین',
  'اردیبهشت',
  'خرداد',
  'تیر',
  'مرداد',
  'شهریور',
  'مهر',
  'آبان',
  'آذر',
  'دی',
  'بهمن',
  'اسفند',
];

const List<String> _persianDigits = [
  '۰',
  '۱',
  '۲',
  '۳',
  '۴',
  '۵',
  '۶',
  '۷',
  '۸',
  '۹',
];

/// Converts Latin digits in [input] to Persian digits.
String toPersianDigits(String input) {
  final buffer = StringBuffer();
  for (final rune in input.runes) {
    final code = rune - 0x30; // ASCII '0'
    buffer.write(
      code >= 0 && code <= 9 ? _persianDigits[code] : String.fromCharCode(rune),
    );
  }
  return buffer.toString();
}

/// Converts Persian digits in [input] back to Latin digits (for parsing).
String fromPersianDigits(String input) {
  final buffer = StringBuffer();
  for (final rune in input.runes) {
    var mapped = false;
    for (var i = 0; i < _persianDigits.length; i++) {
      if (String.fromCharCode(rune) == _persianDigits[i]) {
        buffer.write(i);
        mapped = true;
        break;
      }
    }
    if (!mapped) buffer.writeCharCode(rune);
  }
  return buffer.toString();
}

/// Gregorian instant → Jalali parts.
Jalali jalaliOf(DateTime dt) => Jalali.fromDateTime(dt.toLocal());

/// Jalali parts → Gregorian [DateTime].
DateTime gregorianOf({required int jy, required int jm, required int jd}) =>
    Jalali(jy, jm, jd).toDateTime();

/// `۱۴۰۴/۰۶/۰۱`
String formatJalaliDate(DateTime dt) {
  final j = jalaliOf(dt);
  return toPersianDigits('${j.year}/${_two(j.month)}/${_two(j.day)}');
}

/// Gregorian [DateTime] → ISO-8601 "YYYY-MM-DD" for the wire (the inverse of
/// what [JalaliDatePickerField] emits; API dates stay Gregorian, R6).
String isoDate(DateTime d) =>
    '${d.year.toString().padLeft(4, '0')}-'
    '${d.month.toString().padLeft(2, '0')}-'
    '${d.day.toString().padLeft(2, '0')}';

/// `۱ شهریور ۱۴۰۴`
String formatJalaliLongDate(DateTime dt) {
  final j = jalaliOf(dt);
  return '${toPersianDigits('${j.day}')} ${_jalaliMonthNames[j.month - 1]} '
      '${toPersianDigits('${j.year}')}';
}

/// `۱۴۰۴/۰۶/۰۱ ۱۴:۳۰`
String formatJalaliDateTime(DateTime dt) {
  final local = dt.toLocal();
  return '${formatJalaliDate(local)} ${toPersianDigits(_two(local.hour))}:'
      '${toPersianDigits(_two(local.minute))}';
}

/// Jalali month name for 1–12 (financial report month display).
String jalaliMonthName(int month) => _jalaliMonthNames[month - 1];

String _two(int n) => n.toString().padLeft(2, '0');

/// The single date-entry component of the entire app (plan.md constraint).
///
/// Wraps `showPersianDatePicker`; emits/receives Gregorian [DateTime] so
/// callers never touch the Jalali calendar themselves.
class JalaliDatePickerField extends FormField<DateTime> {
  const JalaliDatePickerField({
    super.key,
    this.label,
    super.initialValue,
    this.firstDate,
    this.lastDate,
    this.onChanged,
    super.validator,
  }) : super(builder: _noBuilder);

  // The state class below owns rendering; FormField only requires a builder.
  static Widget _noBuilder(FormFieldState<DateTime> state) =>
      const SizedBox.shrink();

  /// Field caption shown above the box.
  final String? label;

  /// Inclusive lower bound (Gregorian).
  final DateTime? firstDate;

  /// Inclusive upper bound (Gregorian).
  final DateTime? lastDate;

  /// Called after a successful pick.
  final ValueChanged<DateTime>? onChanged;

  @override
  FormFieldState<DateTime> createState() => _JalaliDatePickerFieldState();
}

class _JalaliDatePickerFieldState extends FormFieldState<DateTime> {
  JalaliDatePickerField get _field => widget as JalaliDatePickerField;

  Future<void> _pick() async {
    final now = Jalali.now();
    final first = _field.firstDate;
    final last = _field.lastDate;
    final firstJ = first == null ? Jalali(1300) : jalaliOf(first);
    final lastJ = last == null ? now.addYears(30) : jalaliOf(last);
    // Empty field → open on Jalali today; never let the plugin fall back
    // to a Gregorian initialDate (mixed-calendar display bug). When a
    // first/last bound exists, clamp the initial date into the allowed
    // range — e.g. a due-date picker bounded by the period end must not
    // open on today when today precedes that bound (assertion crash).
    var initialJ = value == null ? now : jalaliOf(value!);
    if (initialJ.isBefore(firstJ)) initialJ = firstJ;
    if (initialJ.isAfter(lastJ)) initialJ = lastJ;
    final picked = await showPersianDatePicker(
      context: context,
      initialDate: initialJ,
      firstDate: firstJ,
      lastDate: lastJ,
      currentDate: now,
    );
    if (picked != null) {
      didChange(picked.toDateTime());
      _field.onChanged?.call(picked.toDateTime());
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (_field.label != null) ...[
          Text(_field.label!, style: Theme.of(context).textTheme.labelLarge),
          const SizedBox(height: 4),
        ],
        InkWell(
          borderRadius: BorderRadius.circular(8),
          onTap: _pick,
          child: InputDecorator(
            decoration: InputDecoration(
              hintText: l10n.selectDate,
              errorText: errorText,
              suffixIcon: const Icon(Icons.calendar_month_outlined),
            ),
            child: Text(value == null ? '' : formatJalaliDate(value!)),
          ),
        ),
      ],
    );
  }
}
