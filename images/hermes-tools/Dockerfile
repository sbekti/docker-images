FROM python:3.15.2-slim

ENV HERMES_TOOLS_MCP_HOST=0.0.0.0 \
    HERMES_TOOLS_MCP_PORT=9101 \
    PYTHONDONTWRITEBYTECODE=1 \
    PYTHONPATH=/opt/hermes-tools \
    PYTHONUNBUFFERED=1

WORKDIR /opt/hermes-tools

COPY requirements.txt ./

RUN python -m pip install --no-cache-dir --root-user-action=ignore -r requirements.txt \
 && python -m pip check

COPY --chown=10000:10000 hermes_tools_server.py ./
COPY --chown=10000:10000 skills ./skills

RUN python -m compileall -q . \
 && python -c "import hermes_tools_server; assert hermes_tools_server.mcp" \
 && find . -type d -name __pycache__ -prune -exec rm -r {} +

USER 10000:10000

EXPOSE 9101

ENTRYPOINT ["python", "/opt/hermes-tools/hermes_tools_server.py"]
