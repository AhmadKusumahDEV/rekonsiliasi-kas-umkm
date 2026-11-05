"""Encode locally recorded synthetic EFA tutorials; no network access or credentials."""
import argparse
import json
import os
from pathlib import Path
import re
import shutil
import subprocess

root = Path(__file__).resolve().parent.parent
parser = argparse.ArgumentParser()
parser.add_argument('--ffmpeg', default=os.environ.get('FFMPEG') or shutil.which('ffmpeg'))
args = parser.parse_args()
if not args.ffmpeg:
    raise SystemExit('Pass --ffmpeg with the path to an FFmpeg executable.')
manifest_path = root / 'docs-site/video-manifest.json'
manifest = json.loads(manifest_path.read_text(encoding='utf-8'))
output = root / 'docs-site/public/video'
output.mkdir(parents=True, exist_ok=True)

def stamp(seconds):
    return f'{seconds // 3600:02}:{seconds // 60 % 60:02}:{seconds % 60:02}.000'

for item in manifest:
    source = root / '.cache/guide-recordings' / (item['id'] + '.webm')
    probe = subprocess.run([args.ffmpeg, '-i', str(source)], capture_output=True, text=True)
    duration = re.search(r'Duration: (\d+):(\d+):(\d+(?:\.\d+)?)', probe.stderr)
    if not duration:
        raise SystemExit(f'Could not inspect {source}')
    total = int(duration[1])*3600 + int(duration[2])*60 + float(duration[3])
    offset = max(0, total-item['duration'])
    target = output / (item['id'] + '.mp4')
    subprocess.run([args.ffmpeg, '-y', '-loglevel', 'error', '-ss', str(offset), '-i', str(source),
        '-t', str(item['duration']), '-vf', 'fps=20,scale=1280:720', '-an', '-c:v', 'libx264',
        '-preset', 'fast', '-crf', '25', '-pix_fmt', 'yuv420p', '-movflags', '+faststart', str(target)], check=True)
    if target.stat().st_size > 5_000_000:
        raise SystemExit(f'{target.name} exceeds the 5 MB budget')
    item['sizeLabel'] = f"{target.stat().st_size/1_000_000:.1f} MB".replace('.', ',')
    chunks = ['WEBVTT', '']
    for n, scene in enumerate(item['scenes']):
        m,s = map(int, scene['at'].split(':'))
        end = item['scenes'][n+1]['at'] if n+1<len(item['scenes']) else None
        stop = sum(a*b for a,b in zip(map(int,end.split(':')),[60,1])) if end else item['duration']
        chunks.extend([f'{stamp(m*60+s)} --> {stamp(stop)}', scene['caption'], ''])
    (output / (item['id']+'.vtt')).write_text('\n'.join(chunks).rstrip()+'\n', encoding='utf-8')
    print(f"{target.name}: {item['duration']} seconds, {item['sizeLabel']}, 1280x720, silent")
manifest_path.write_text(json.dumps(manifest, ensure_ascii=False, indent=2)+'\n', encoding='utf-8')
