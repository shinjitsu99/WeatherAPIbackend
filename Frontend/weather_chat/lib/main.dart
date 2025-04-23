import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'dart:async';

void main() => runApp(WeatherChatApp());

class WeatherChatApp extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'WeatherBot',
      theme: ThemeData(primarySwatch: Colors.blue),
      home: WeatherChatScreen(),
      debugShowCheckedModeBanner: false,
    );
  }
}

class WeatherChatScreen extends StatefulWidget {
  @override
  _WeatherChatScreenState createState() => _WeatherChatScreenState();
}

class _WeatherChatScreenState extends State<WeatherChatScreen> {
  final TextEditingController _controller = TextEditingController();
  final TextEditingController _feedbackController = TextEditingController();
  final List<Map<String, String>> _messages = [];
  String? _currentOriginalQuery;
  String? _currentResponse;
  bool? _disliked;

  Future<void> _sendQuery() async {
    final query = _controller.text.trim();
    if (query.isEmpty) return;

    setState(() {
      _messages.add({'user': query});
      _controller.clear();
    });

    try {
      final url = Uri.parse('http://127.0.0.1:3000/query');
      final response = await http.post(
        url,
        headers: {'Content-Type': 'application/json'},
        body: jsonEncode({'original_query': query}),
      ).timeout(const Duration(seconds: 10));

      if (response.statusCode == 200) {
        final data = jsonDecode(response.body);
        setState(() {
          _messages.add({'bot': data['translated_response']});
          _currentOriginalQuery = data['original_query'];
          _currentResponse = data['translated_response'];
          _disliked = null;
        });
      } else {
        setState(() {
          _messages.add({'bot': 'Error processing request'});
        });
      }
    } catch (e) {
      setState(() {
        _messages.add({'bot': 'Network error'});
      });
    }
  }

  Future<void> _sendFeedback() async {
    if (_currentOriginalQuery == null || _currentResponse == null) return;

    final feedback = _feedbackController.text.trim();
    if (_disliked == true && feedback.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Please provide feedback')),
      );
      return;
    }

    try {
      final url = Uri.parse('http://127.0.0.1:3000/feedback');
      final response = await http.put(
        url,
        headers: {'Content-Type': 'application/json'},
        body: jsonEncode({
          'original_query': _currentOriginalQuery,
          'feedback': feedback,
          'disliked': _disliked.toString(),
        }),
      ).timeout(const Duration(seconds: 10));

      if (response.statusCode == 200) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Feedback saved!')),
        );
        _feedbackController.clear();
      }
    } catch (e) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Failed to save feedback')),
      );
    }
  }

  void _handleReaction(bool isLike) {
    setState(() {
      _disliked = !isLike;
    });

    if (isLike) {
      _sendFeedback(); // Send empty feedback for likes
    }
  }

  Widget _buildMessage(Map<String, String> msg) {
    final isUser = msg.containsKey('user');
    final text = isUser ? msg['user']! : msg['bot']!;
    final align = isUser ? CrossAxisAlignment.end : CrossAxisAlignment.start;
    final bgColor = isUser ? Colors.blue[100] : Colors.grey[300];

    return Column(
      crossAxisAlignment: align,
      children: [
        Container(
          padding: const EdgeInsets.all(12),
          margin: const EdgeInsets.symmetric(vertical: 4, horizontal: 12),
          decoration: BoxDecoration(
            color: bgColor,
            borderRadius: BorderRadius.circular(12),
          ),
          child: Text(text),
        ),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('WeatherBot')),
      body: Column(
        children: [
          Expanded(
            child: ListView.builder(
              padding: const EdgeInsets.symmetric(vertical: 8),
              itemCount: _messages.length,
              itemBuilder: (context, index) => _buildMessage(_messages[index]),
            ),
          ),
          if (_currentResponse != null) ...[
            Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                IconButton(
                  icon: Icon(Icons.thumb_up, 
                    color: _disliked == false ? Colors.green : Colors.grey),
                  onPressed: () => _handleReaction(true),
                ),
                IconButton(
                  icon: Icon(Icons.thumb_down, 
                    color: _disliked == true ? Colors.red : Colors.grey),
                  onPressed: () => _handleReaction(false),
                ),
              ],
            ),
            if (_disliked == true)
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 12),
                child: Row(
                  children: [
                    Expanded(
                      child: TextField(
                        controller: _feedbackController,
                        decoration: const InputDecoration(
                          hintText: 'What went wrong?',
                          border: OutlineInputBorder(),
                        ),
                      ),
                    ),
                    IconButton(
                      icon: const Icon(Icons.send),
                      onPressed: _sendFeedback,
                    ),
                  ],
                ),
              ),
          ],
          Padding(
            padding: const EdgeInsets.all(8.0),
            child: Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: _controller,
                    onSubmitted: (_) => _sendQuery(),
                    decoration: const InputDecoration(
                      hintText: 'Ask about the weather...',
                      border: OutlineInputBorder(),
                    ),
                  ),
                ),
                IconButton(
                  icon: const Icon(Icons.send),
                  onPressed: _sendQuery,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}