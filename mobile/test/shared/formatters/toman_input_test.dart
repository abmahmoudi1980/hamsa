import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:hamsa/shared/formatters/money_text.dart';
import 'package:hamsa/shared/formatters/toman_input.dart';

void main() {
  const fmt = TomanInputFormatter();

  TextEditingValue edit(
    String oldText,
    String newText, {
    int? base,
  }) {
    return fmt.formatEditUpdate(
      TextEditingValue(text: oldText),
      TextEditingValue(
        text: newText,
        selection: TextSelection.collapsed(
          offset: base ?? newText.length,
        ),
      ),
    );
  }

  group('TomanInputFormatter', () {
    test('groups digits in threes with Persian separator while typing', () {
      var text = '';
      for (final d in '2600000'.split('')) {
        text = edit(text, text + d).text;
      }
      expect(text, '۲٬۶۰۰٬۰۰۰');
    });

    test('keeps separators when more digits arrive', () {
      expect(edit('۲٬۶۰۰٬۰۰۰', '26000000').text, '۲۶٬۰۰۰٬۰۰۰');
    });

    test('accepts Latin digits and converts to Persian', () {
      expect(edit('', '1234').text, '۱٬۲۳۴');
    });

    test('drops non-digit characters (paste noise)', () {
      expect(edit('', ' 12abc34 ').text, '۱٬۲۳۴');
    });
    test('preserves caret when inserting at the middle', () {
      // Caret after the 2nd digit of "1500"; typing 9 → "15900".
      final result = edit('۱٬۵۰۰', '۱٬۵۹۰۰', base: 4);
      expect(result.text, '۱۵٬۹۰۰');
      // Caret sits right after the newly typed digit (the ۹).
      expect(result.selection.baseOffset, 4);
    });

    test('sequential typing keeps digits in order (caret regression)', () {
      var value = const TextEditingValue();
      for (final d in '1500000'.split('')) {
        final newText = value.text + d;
        value = fmt.formatEditUpdate(
          value,
          TextEditingValue(
            text: newText,
            selection: TextSelection.collapsed(offset: newText.length),
          ),
        );
      }
      expect(value.text, '۱٬۵۰۰٬۰۰۰');
      expect(value.selection.baseOffset, value.text.length);
    });

    test('allows clearing back to empty', () {
      expect(edit('۲٬۶۰۰', '').text, '');
    });
  });

  group('parseToman', () {
    test('round-trips formatted output', () {
      expect(parseToman(formatToman(2600000)), 2600000);
    });

    test('accepts Persian digits and both separators', () {
      expect(parseToman('۲٬۶۰۰٬۰۰۰'), 2600000);
      expect(parseToman('2,600,000'), 2600000);
      expect(parseToman('2600000'), 2600000);
    });

    test('rejects non-numeric input', () {
      expect(parseToman(''), isNull);
      expect(parseToman('abc'), isNull);
    });
  });

  test('formatToman renders Persian digits with separators', () {
    const lri = '\u2066', pdi = '\u2069';
    expect(formatToman(2600000), '$lri۲٬۶۰۰٬۰۰۰$pdi');
    expect(formatToman(999), '$lri۹۹۹$pdi');
    // Negative: minus stays attached inside the isolate (bidi regression).
    expect(formatToman(-5700515), '$lri-۵٬۷۰۰٬۵۱۵$pdi');
  });
}
