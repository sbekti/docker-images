FROM nousresearch/hermes-agent:v2026.7.20

ENV HERMES_TOOLS_MCP_HOST=0.0.0.0 \
    HERMES_TOOLS_MCP_PORT=9101 \
    PYTHONDONTWRITEBYTECODE=1 \
    PYTHONPATH=/opt/hermes-tools \
    PYTHONUNBUFFERED=1

USER root
WORKDIR /opt/hermes-tools

COPY --chown=10000:10000 hermes_tools_server.py /opt/hermes-tools/hermes_tools_server.py
COPY --chown=10000:10000 skills /opt/hermes-tools/skills

RUN /opt/hermes/.venv/bin/python -m compileall -q /opt/hermes-tools \
 && /opt/hermes/.venv/bin/python -c "import hermes_tools_server; assert hermes_tools_server.mcp" \
 && find /opt/hermes-tools -type d -name __pycache__ -prune -exec rm -rf {} + \
 && chmod -R a=rX /opt/hermes-tools

USER 10000:10000

EXPOSE 9101

ENTRYPOINT ["/opt/hermes/.venv/bin/python", "/opt/hermes-tools/hermes_tools_server.py"]
CMD []
