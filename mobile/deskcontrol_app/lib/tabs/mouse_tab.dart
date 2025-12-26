import 'dart:async';
import 'package:flutter/material.dart';
import '../desk_socket.dart';

class MouseTab extends StatefulWidget {
  final DeskSocket desk;
  final double sensitivity;
  final double scrollSpeed;

  // ✅ nuevo
  final int holdDelayMs;

  const MouseTab({
    super.key,
    required this.desk,
    required this.sensitivity,
    required this.scrollSpeed,
    required this.holdDelayMs,
  });

  @override
  State<MouseTab> createState() => _MouseTabState();
}

class _MouseTabState extends State<MouseTab> {
  int? _activePointer;
  Offset? _downPos;
  Offset? _lastPos;

  Timer? _holdTimer;
  bool _holding = false;
  bool _moved = false;

  // Si se mueve más que esto ANTES de que se cumpla holdDelay => no se hace hold
  static const double _cancelHoldMovePx = 6.0;

  void _sendMove(double dx, double dy) {
    widget.desk.send({
      "type": "mouse_move",
      "dx": (dx * widget.sensitivity).round(),
      "dy": (dy * widget.sensitivity).round(),
    });
  }

  void _click(String button) => widget.desk.send({"type": "mouse_click", "button": button});
  void _down(String button) => widget.desk.send({"type": "mouse_down", "button": button});
  void _up(String button) => widget.desk.send({"type": "mouse_up", "button": button});

  void _scroll(double dy) {
    widget.desk.send({
      "type": "mouse_scroll",
      "dy": (dy * 120 * widget.scrollSpeed).round(),
    });
  }

  void _startHoldTimer() {
    _holdTimer?.cancel();
    final ms = widget.holdDelayMs.clamp(50, 2000);
    _holdTimer = Timer(Duration(milliseconds: ms), () {
      // si sigue tocando y no se canceló por movimiento
      _holding = true;
      _down("left");
    });
  }

  void _cancelHoldTimer() {
    _holdTimer?.cancel();
    _holdTimer = null;
  }

  void _releaseHoldIfNeeded() {
    if (_holding) {
      _holding = false;
      _up("left");
    }
  }

  @override
  void dispose() {
    _cancelHoldTimer();
    _releaseHoldIfNeeded();
    super.dispose();
  }

  Widget _buildTouchpad() {
    return Container(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(18),
        border: Border.all(width: 2),
      ),
      child: Listener(
        behavior: HitTestBehavior.opaque,

        onPointerDown: (e) {
          if (_activePointer != null) return;

          _activePointer = e.pointer;
          _downPos = e.localPosition;
          _lastPos = e.localPosition;

          _moved = false;
          _holding = false;

          _startHoldTimer();
        },

        onPointerMove: (e) {
          if (_activePointer != e.pointer) return;

          final last = _lastPos;
          final down = _downPos;
          if (last == null || down == null) return;

          final now = e.localPosition;
          _lastPos = now;

          // distancia desde el punto inicial
          final dist = (now - down).distance;

          // si se movió, ya no es tap
          if (dist > 0.5) _moved = true;

          // ✅ si aún NO está en hold y se movió más que X px,
          // cancelamos el hold para permitir “solo mover”
          if (!_holding && _holdTimer != null && dist >= _cancelHoldMovePx) {
            _cancelHoldTimer();
          }

          // siempre mover (si está hold, esto arrastra)
          final dx = now.dx - last.dx;
          final dy = now.dy - last.dy;
          _sendMove(dx, dy);
        },

        onPointerUp: (e) {
          if (_activePointer != e.pointer) return;

          _cancelHoldTimer();

          // si NO fue hold y casi no movió => click normal
          if (!_holding && !_moved) {
            _click("left");
          }

          // si fue hold => soltar
          _releaseHoldIfNeeded();

          _activePointer = null;
          _downPos = null;
          _lastPos = null;
          _moved = false;
        },

        onPointerCancel: (e) {
          if (_activePointer != e.pointer) return;

          _cancelHoldTimer();
          _releaseHoldIfNeeded();

          _activePointer = null;
          _downPos = null;
          _lastPos = null;
          _moved = false;
        },

        child: Center(
          child: Text(
            "TOUCHPAD\n\n"
            "- Toca: click\n"
            "- Arrastra: mover\n"
            "- Mantén (${"${widget.holdDelayMs}ms"}): click sostenido (drag)\n"
            "- Scroll: barra derecha",
            textAlign: TextAlign.center,
          ),
        ),
      ),
    );
  }

  Widget _buildScrollBar() {
    return Container(
      width: 56,
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(18),
        border: Border.all(width: 2),
      ),
      child: Listener(
        behavior: HitTestBehavior.opaque,
        onPointerMove: (e) {
          _scroll(-e.delta.dy / 20.0);
        },
        child: const Center(
          child: RotatedBox(
            quarterTurns: 3,
            child: Text("SCROLL", textAlign: TextAlign.center),
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          children: [
            Expanded(
              child: Row(
                children: [
                  Expanded(child: _buildTouchpad()),
                  const SizedBox(width: 10),
                  _buildScrollBar(),
                ],
              ),
            ),
            const SizedBox(height: 10),
            Row(
              children: [
                Expanded(
                  child: ElevatedButton(
                    onPressed: () => _click("left"),
                    child: const Text("Click"),
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: ElevatedButton(
                    onPressed: () => _click("right"),
                    child: const Text("Der"),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
