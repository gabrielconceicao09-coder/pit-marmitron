// lib/screens/client/qr_scanner_screen.dart
//
// QR scanner for OTP unlock. mobile_scanner ^5.0.0 compliant.
//
// Camera readiness + torch state are observed through the controller's
// ValueNotifier<MobileScannerState>; toggleTorch()/switchCamera() are guarded
// by state.isRunning so they are never called before the session is live.
// ignore_for_file: prefer_const_constructors

import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:mobile_scanner/mobile_scanner.dart';
import '../../theme/app_theme.dart';

class QrScannerScreen extends StatefulWidget {
  final String expectedCode;

  const QrScannerScreen({super.key, required this.expectedCode});

  @override
  State<QrScannerScreen> createState() => _QrScannerScreenState();
}

class _QrScannerScreenState extends State<QrScannerScreen> {
  late final MobileScannerController _scannerCtrl;

  // Double-pop guard: MLKit on Android fires onDetect multiple times per
  // burst. Once we have a valid code and have called Navigator.pop(), we
  // must ignore every subsequent detection callback.
  bool _scanned = false;

  @override
  void initState() {
    super.initState();
    _scannerCtrl = MobileScannerController(
      formats: [BarcodeFormat.qrCode],
      facing: CameraFacing.back,
    );
  }

  @override
  void dispose() {
    // stop() then dispose(): stop() releases the hardware camera lock, then
    // dispose() frees the Dart-side controller. dispose() without stop() can
    // leave the camera acquired on some Android devices.
    _scannerCtrl.stop();
    _scannerCtrl.dispose();
    super.dispose();
  }

  void _toggleTorch() {
    if (!_scannerCtrl.value.isRunning) return;
    _scannerCtrl.toggleTorch();
  }

  void _switchCamera() {
    if (!_scannerCtrl.value.isRunning) return;
    _scannerCtrl.switchCamera();
  }

  void _onDetect(BarcodeCapture capture) {
    if (_scanned) return;

    for (final barcode in capture.barcodes) {
      final raw = barcode.rawValue;
      if (raw == null) continue;

      final isValidOtp = raw == widget.expectedCode;

      if (isValidOtp) {
        _scanned = true;
        Navigator.pop(context, raw);
        return;
      }

      // Invalid QR content — non-blocking snackbar, scanner keeps running.
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(
              'QR Code inválido. Não corresponde a este pedido.',
              style: GoogleFonts.dmSans(fontSize: 13, color: Colors.white),
            ),
            backgroundColor: Colors.red.shade700,
            behavior: SnackBarBehavior.floating,
            shape:
                RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
            margin: const EdgeInsets.all(16),
            duration: const Duration(seconds: 2),
          ),
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.black,
      appBar: AppBar(
        backgroundColor: Colors.black,
        foregroundColor: Colors.white,
        title: Text(
          'Escanear QR Code',
          style: GoogleFonts.spaceGrotesk(
            fontSize: 17,
            fontWeight: FontWeight.w600,
            color: Colors.white,
          ),
        ),
        actions: [
          // ── Torch button ────────────────────────────────────────────────
          ValueListenableBuilder<MobileScannerState>(
            valueListenable: _scannerCtrl,
            builder: (context, state, _) {
              final isRunning = state.isRunning;
              final torchOn = state.torchState == TorchState.on;

              return IconButton(
                icon: Icon(
                  torchOn
                      ? Icons.flash_on_rounded
                      : Icons.flash_off_rounded,
                  color: isRunning
                      ? (torchOn ? Colors.yellow : Colors.white)
                      : Colors.white38,
                ),
                tooltip: torchOn ? 'Desligar lanterna' : 'Ligar lanterna',
                onPressed: isRunning ? _toggleTorch : null,
              );
            },
          ),

          // ── Camera switch button ─────────────────────────────────────────
          ValueListenableBuilder<MobileScannerState>(
            valueListenable: _scannerCtrl,
            builder: (context, state, _) {
              return IconButton(
                icon: Icon(
                  Icons.flip_camera_ios_rounded,
                  color: state.isRunning ? Colors.white : Colors.white38,
                ),
                tooltip: 'Trocar câmera',
                onPressed: state.isRunning ? _switchCamera : null,
              );
            },
          ),
        ],
      ),
      body: Stack(
        children: [
          // ── Camera feed ──────────────────────────────────────────────────
          MobileScanner(
            controller: _scannerCtrl,
            onDetect: _onDetect,
          ),

          // ── Frosted overlay with cutout ──────────────────────────────────
          Positioned.fill(
            child: CustomPaint(
              painter: _ScannerOverlayPainter(
                cutoutSize: 240,
                borderColor: AppColors.accent,
                overlayColor: Colors.black.withValues(alpha: 0.55),
              ),
            ),
          ),

          // ── Camera initialising indicator ────────────────────────────────
          ValueListenableBuilder<MobileScannerState>(
            valueListenable: _scannerCtrl,
            builder: (context, state, _) {
              if (state.isRunning) return const SizedBox.shrink();
              return Positioned.fill(
                child: Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const SizedBox(
                        width: 32,
                        height: 32,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: AppColors.accent,
                        ),
                      ),
                      const SizedBox(height: 12),
                      Text(
                        'Iniciando câmera...',
                        style: GoogleFonts.dmSans(
                            fontSize: 13, color: Colors.white70),
                      ),
                    ],
                  ),
                ),
              );
            },
          ),

          // ── Instructions + manual entry fallback ─────────────────────────
          Positioned(
            bottom: 80,
            left: 32,
            right: 32,
            child: Column(
              children: [
                Container(
                  padding: const EdgeInsets.symmetric(
                      horizontal: 16, vertical: 10),
                  decoration: BoxDecoration(
                    color: Colors.black.withValues(alpha: 0.6),
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Text(
                    'Aponte para o QR Code no painel do robô',
                    textAlign: TextAlign.center,
                    style: GoogleFonts.dmSans(
                      fontSize: 13,
                      color: Colors.white,
                      height: 1.4,
                    ),
                  ),
                ),
                const SizedBox(height: 16),
                TextButton(
                  onPressed: () => Navigator.pop(context, null),
                  child: Text(
                    'Digitar código manualmente',
                    style: GoogleFonts.dmSans(
                      fontSize: 13,
                      color: Colors.white60,
                      decoration: TextDecoration.underline,
                      decorationColor: Colors.white38,
                    ),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

// ─── SCANNER OVERLAY PAINTER ─────────────────────────────────────────────────

class _ScannerOverlayPainter extends CustomPainter {
  final double cutoutSize;
  final Color borderColor;
  final Color overlayColor;

  const _ScannerOverlayPainter({
    required this.cutoutSize,
    required this.borderColor,
    required this.overlayColor,
  });

  @override
  void paint(Canvas canvas, Size size) {
    final cx = size.width / 2;
    final cy = size.height / 2;
    final half = cutoutSize / 2;

    final cutout = Rect.fromLTWH(cx - half, cy - half, cutoutSize, cutoutSize);
    final cutoutRRect =
        RRect.fromRectAndRadius(cutout, const Radius.circular(16));

    final overlayPath = Path()
      ..addRect(Rect.fromLTWH(0, 0, size.width, size.height))
      ..addRRect(cutoutRRect)
      ..fillType = PathFillType.evenOdd;

    canvas.drawPath(overlayPath, Paint()..color = overlayColor);

    final bracketPaint = Paint()
      ..color = borderColor
      ..strokeWidth = 3.5
      ..strokeCap = StrokeCap.round
      ..style = PaintingStyle.stroke;

    const bracketLen = 28.0;
    const r = 16.0;

    canvas.drawLine(Offset(cutout.left + r, cutout.top),
        Offset(cutout.left + r + bracketLen, cutout.top), bracketPaint);
    canvas.drawLine(Offset(cutout.left, cutout.top + r),
        Offset(cutout.left, cutout.top + r + bracketLen), bracketPaint);
    canvas.drawLine(Offset(cutout.right - r, cutout.top),
        Offset(cutout.right - r - bracketLen, cutout.top), bracketPaint);
    canvas.drawLine(Offset(cutout.right, cutout.top + r),
        Offset(cutout.right, cutout.top + r + bracketLen), bracketPaint);
    canvas.drawLine(Offset(cutout.left + r, cutout.bottom),
        Offset(cutout.left + r + bracketLen, cutout.bottom), bracketPaint);
    canvas.drawLine(Offset(cutout.left, cutout.bottom - r),
        Offset(cutout.left, cutout.bottom - r - bracketLen), bracketPaint);
    canvas.drawLine(Offset(cutout.right - r, cutout.bottom),
        Offset(cutout.right - r - bracketLen, cutout.bottom), bracketPaint);
    canvas.drawLine(Offset(cutout.right, cutout.bottom - r),
        Offset(cutout.right, cutout.bottom - r - bracketLen), bracketPaint);
  }

  @override
  bool shouldRepaint(_ScannerOverlayPainter old) =>
      old.cutoutSize != cutoutSize ||
      old.borderColor != borderColor ||
      old.overlayColor != overlayColor;
}
