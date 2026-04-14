#!/usr/bin/env bash
branch=$(git symbolic-ref HEAD 2>/dev/null | sed 's|refs/heads/||')
if [ "$branch" = "main" ]; then
  echo "直接 main へのプッシュは禁止されています。"
  exit 1
fi
exit 0
