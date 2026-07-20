import 'dart:async';
import 'package:flutter/material.dart';
import '../desk_socket.dart';

class MouseTab extends StatefulWidget {
  final DeskSocket desk;
  final double sensitivity;
  final double scrollSpeed;
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

  static const double _cancelHoldMovePx = 6.0;

  void _sendMove(double dx, double dy) {
    widget.desk.send({
      "type": "mouse_move",
      "dx": (dx * widget.sensitivity).round(),
      "dy": (dy * widget.sensitivity).round(),
    });
  }

  void _click(String button) =>
      widget.desk.send({"type": "mouse_click", "button": button});
  void _down(String button) =>
      widget.desk.send({"type": "mouse_down", "button": button});
  void _up(String button) =>
      widget.desk.send({"type": "mouse_up", "button": button});

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
          final dist = (now - down).distance;
          if (dist > 0.5) _moved = true;
          if (!_holding && _holdTimer != null && dist >= _cancelHoldMovePx) {
            _cancelHoldTimer();
          }
          final dx = now.dx - last.dx;
          final dy = now.dy - last.dy;
          _sendMove(dx, dy);
        },
        onPointerUp: (e) {
          if (_activePointer != e.pointer) return;
          _cancelHoldTimer();
          if (!_holding && !_moved) _click("left");
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
            "- Mantén (${widget.holdDelayMs}ms): drag\n"
            "- Scroll: barra lateral",
            textAlign: TextAlign.center,
          ),
        ),
      ),
    );
  }

  Widget _buildScrollBar({bool vertical = true}) {
    return Container(
      width: vertical ? 56 : double.infinity,
      height: vertical ? double.infinity : 56,
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(18),
        border: Border.all(width: 2),
      ),
      child: Listener(
        behavior: HitTestBehavior.opaque,
        onPointerMove: (e) {
          final delta = vertical ? -e.delta.dy / 20.0 : -e.delta.dx / 20.0;
          _scroll(delta);
        },
        child: Center(
          child: RotatedBox(
            quarterTurns: vertical ? 3 : 0,
            child: const Text("SCROLL", textAlign: TextAlign.center),
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final isLandscape =
        MediaQuery.of(context).orientation == Orientation.landscape;

    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.all(8),
        child: isLandscape ? _buildLandscape() : _buildPortrait(),
      ),
    );
  }

  Widget _buildPortrait() {
    return Column(
      children: [
        Expanded(
          child: Row(
            children: [
              Expanded(child: _buildTouchpad()),
              const SizedBox(width: 8),
              _buildScrollBar(vertical: true),
            ],
          ),
        ),
        const SizedBox(height: 8),
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
    );
  }

  Widget _buildLandscape() {
    return Column(
      children: [
        Expanded(
          child: Row(
            children: [
              Expanded(child: _buildTouchpad()),
              const SizedBox(width: 8),
              SizedBox(
                width: 80,
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Expanded(
                      child: SizedBox(
                        width: double.infinity,
                        child: ElevatedButton(
                          onPressed: () => _click("left"),
                          child: const Text("Click"),
                        ),
                      ),
                    ),
                    const SizedBox(height: 8),
                    Expanded(
                      child: SizedBox(
                        width: double.infinity,
                        child: ElevatedButton(
                          onPressed: () => _click("right"),
                          child: const Text("Der"),
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 8),
        SizedBox(
          height: 48,
          child: _buildScrollBar(vertical: false),
        ),
      ],
    );
  }
}
