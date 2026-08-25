import type { StreamFrame } from "../types";

export interface RendererStats {
  decoded: number;
  decodeErrors: number;
  dropped: number;
  width: number;
  height: number;
  lastFrameAt: number; // epoch ms of last decoded frame
}

type VideoDecoderLike = {
  decodeQueueSize: number;
  state: string;
  configure(config: { codec: string; optimizeForLatency?: boolean }): void;
  decode(chunk: unknown): void;
  reset(): void;
  close(): void;
};

const MAX_QUEUE = 8;

export class H264Renderer {
  private canvas: HTMLCanvasElement | null = null;
  private context: CanvasRenderingContext2D | null = null;
  private decoder: VideoDecoderLike | null = null;
  private decoderBroken = false; // recreate on next keyframe
  private fallbackTick = 0;

  stats: RendererStats = { decoded: 0, decodeErrors: 0, dropped: 0, width: 0, height: 0, lastFrameAt: 0 };
  onStats: ((stats: RendererStats) => void) | null = null;
  private lastStatsEmit = 0;

  attach(canvas: HTMLCanvasElement | null) {
    this.canvas = canvas;
    this.context = canvas?.getContext("2d") ?? null;
    if (!canvas) {
      this.decoder?.close();
      this.decoder = null;
      return;
    }

    this.drawIdle();
  }

  push(frame: StreamFrame) {
    if (!this.canvas || !this.context) {
      return;
    }

    const bytes = base64ToBytes(frame.data);

    // Backpressure: drop stale deltas instead of growing the queue.
    if (this.decoder && !this.decoderBroken && this.decoder.decodeQueueSize > MAX_QUEUE && !frame.keyFrame) {
      this.stats.dropped += 1;
      this.emitStats();
      return;
    }

    // (Re)create the decoder lazily — only at a keyframe so it can start clean.
    if (!this.decoder || (this.decoderBroken && frame.keyFrame)) {
      if (this.decoder) {
        this.decoder.close();
        this.decoder = null;
      }
      this.decoderBroken = false;
      if (!this.prepareDecoder()) {
        this.drawFallback(frame);
        return;
      }
    }

    try {
      const EncodedVideoChunkCtor = webCodecs().EncodedVideoChunk;
      if (!EncodedVideoChunkCtor || !this.decoder) {
        throw new Error("EncodedVideoChunk unavailable");
      }
      const decoder = this.decoder;
      // A delta frame arriving while broken/unconfigured would poison the
      // stream — wait for the next keyframe instead.
      if (decoder.state !== "configured" && !frame.keyFrame) {
        return;
      }
      decoder.decode(
        new EncodedVideoChunkCtor({
          type: frame.keyFrame ? "key" : "delta",
          timestamp: Math.floor(frame.timestampNs / 1000),
          data: bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer
        })
      );
    } catch (err) {
      this.stats.decodeErrors += 1;
      // Queue-full and transient errors must not kill the pipeline forever:
      // mark broken and rebuild from the next keyframe.
      this.decoderBroken = true;
      logThrottled("decoder error", err);
      this.emitStats();
    }
  }

  private prepareDecoder(): boolean {
    const VideoDecoderCtor = webCodecs().VideoDecoder;
    if (!VideoDecoderCtor) {
      return false;
    }

    this.decoder = new VideoDecoderCtor({
      output: (videoFrame: VideoFrameLike) => {
        if (!this.canvas || !this.context) {
          videoFrame.close();
          return;
        }
        if (videoFrame.displayWidth > 0) {
          this.stats.width = videoFrame.displayWidth;
          this.stats.height = videoFrame.displayHeight;
        }
        // Canvas backing store follows the stream once real dimensions land.
        if ((this.canvas.width !== videoFrame.displayWidth || this.canvas.height !== videoFrame.displayHeight) && videoFrame.displayWidth > 0) {
          this.canvas.width = videoFrame.displayWidth;
          this.canvas.height = videoFrame.displayHeight;
        }
        this.context.drawImage(videoFrame as unknown as CanvasImageSource, 0, 0);
        videoFrame.close();
        this.stats.decoded += 1;
        this.stats.lastFrameAt = Date.now();
        this.emitStats();
      },
      error: (err: Error) => {
        this.stats.decodeErrors += 1;
        this.decoderBroken = true;
        logThrottled("decoder fatal", err);
        this.emitStats();
      }
    });
    this.decoder.configure({ codec: "avc1.42E01E", optimizeForLatency: true });
    return true;
  }

  private emitStats(force = false) {
    // Throttle UI pushes — frames arrive up to 60/s, panels need far less.
    const now = Date.now();
    if (!force && now - this.lastStatsEmit < 500) {
      return;
    }
    this.lastStatsEmit = now;
    this.onStats?.({ ...this.stats });
  }

  private drawIdle() {
    if (!this.canvas || !this.context) {
      return;
    }
    const { width, height } = this.canvas;
    this.context.fillStyle = "#0b0c0f";
    this.context.fillRect(0, 0, width, height);
    this.context.strokeStyle = "#242a31";
    this.context.lineWidth = 1;
    for (let x = 0; x < width; x += 32) {
      this.context.beginPath();
      this.context.moveTo(x, 0);
      this.context.lineTo(x, height);
      this.context.stroke();
    }
  }

  private drawFallback(frame: StreamFrame) {
    if (!this.canvas || !this.context) {
      return;
    }
    const { width, height } = this.canvas;
    this.fallbackTick = (this.fallbackTick + 1) % Math.max(width, 1);

    this.context.fillStyle = "#0b0c0f";
    this.context.fillRect(0, 0, width, height);
    this.context.fillStyle = "#151922";
    this.context.fillRect(0, 0, width, height);
    this.context.fillStyle = frame.keyFrame ? "#4fd18b" : "#66d9ef";
    this.context.fillRect(this.fallbackTick, 0, 4, height);
    this.context.fillStyle = "#d6dde8";
    this.context.font = "14px Consolas, monospace";
    this.context.fillText(`H264 ${frame.keyFrame ? "key" : "delta"} frame received`, 24, 32);
  }
}

interface VideoFrameLike {
  close(): void;
  displayWidth: number;
  displayHeight: number;
}

let lastLogAt = 0;
function logThrottled(label: string, err: unknown) {
  const now = Date.now();
  if (now - lastLogAt < 2000) {
    return;
  }
  lastLogAt = now;
  console.warn(`[H264] ${label}:`, err);
}

function base64ToBytes(base64: string): Uint8Array {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index);
  }
  return bytes;
}

function webCodecs(): {
  VideoDecoder?: new (init: {
    output(frame: VideoFrameLike): void;
    error(error: Error): void;
  }) => VideoDecoderLike;
  EncodedVideoChunk?: new (init: {
    type: "key" | "delta";
    timestamp: number;
    data: BufferSource;
  }) => unknown;
} {
  return window as unknown as {
    VideoDecoder?: new (init: {
      output(frame: VideoFrameLike): void;
      error(error: Error): void;
    }) => VideoDecoderLike;
    EncodedVideoChunk?: new (init: {
      type: "key" | "delta";
      timestamp: number;
      data: BufferSource;
    }) => unknown;
  };
}
