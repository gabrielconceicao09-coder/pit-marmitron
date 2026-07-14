import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:unbot/theme/app_theme.dart';

void main() {
  const fallback = Color(0xFFF5F5F5);

  test('parses a valid 6-digit hex', () {
    expect(hexBg('003366'), const Color(0xFF003366));
  });

  test('accepts an optional leading #', () {
    expect(hexBg('#00C97A'), const Color(0xFF00C97A));
  });

  test('is always fully opaque', () {
    expect(hexBg('FFFFFF').a, 1.0);
  });

  // These inputs previously crashed build() via int.parse (MB-A2); now they
  // must fall back instead of throwing.
  test('falls back on malformed / missing values', () {
    for (final bad in ['', 'nothex', 'F5F5', 'F5F5F5F5', '   ', 'GGGGGG']) {
      expect(hexBg(bad), fallback, reason: 'input: "$bad"');
    }
  });
}
