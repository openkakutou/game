# Recording image: the native toolchain image (Xvfb, xdotool, SDL2, Go) plus
# ffmpeg and the libraries Ikemen GO's Linux build loads.
ARG BASE
FROM ${BASE}
RUN apt-get update && apt-get install -y --no-install-recommends \
      ffmpeg libgl1 libglx-mesa0 libasound2 libpulse0 libgtk-3-0 libopenal1 libglfw3 \
    && rm -rf /var/lib/apt/lists/*
