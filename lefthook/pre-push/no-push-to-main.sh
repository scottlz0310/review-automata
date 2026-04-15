#!/usr/bin/env bash
branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null)
if [ "$branch" = "main" ]; then
  echo "直接 main へのプッシュは禁止されています。"
  exit 1
fi
exit 0
