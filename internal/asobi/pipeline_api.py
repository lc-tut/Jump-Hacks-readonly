from flask import Flask, request, jsonify
import subprocess

app = Flask(__name__)

@app.route('/run_pipeline', methods=['POST'])
def run_pipeline():
    try:
        # 必要ならリクエストからパラメータ取得
        # data = request.get_json()
        # pages = data.get("pages")
        # key = data.get("key")

        # スクリプト実行
        result = subprocess.run(
            ["/bin/bash", "/workspace/internal/asobi/run_custom_pipeline.sh"],
            capture_output=True, text=True
        )
        return jsonify({
            "stdout": result.stdout,
            "stderr": result.stderr,
            "returncode": result.returncode
        }), 200 if result.returncode == 0 else 500
    except Exception as e:
        return jsonify({"error": str(e)}), 500

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5001)