import requests
import ctypes
import os
from sys import platform
from platform import machine
import json
import psutil
import time
from glob import glob
# from readability import Document

if platform == 'darwin':
    file_ext = '-arm64.dylib' if machine() == "arm64" else '-x86.dylib'
elif platform in ('win32', 'cygwin'):
    file_ext = '-win64.dll' if 8 == ctypes.sizeof(ctypes.c_voidp) else '-win32.dll'
else:
    if machine() == "aarch64":
        file_ext = '-arm64.so'
    elif "x86" in machine():
        file_ext = '-x86.so'
    else:
        file_ext = '-amd64.so'


root_dir = os.path.abspath(os.path.dirname(__file__))
library = ctypes.cdll.LoadLibrary(f'{root_dir}/libparser{file_ext}')

parse = library.parse
parse.argtypes = [ctypes.c_char_p]
parse.restype = ctypes.c_char_p

freeMemory = library.freeMemory
freeMemory.argtypes = [ctypes.c_char_p]

def parse_html(html_content, url):
    result = parse(html_content.encode('utf-8'), url.encode('utf-8'))
    data = json.loads(ctypes.c_char_p(result).value.decode('utf-8'))
    freeMemory(data["id"].encode('utf-8'))
    return data

url = "https://example.com"

idx = 0
s = time.time()
process = psutil.Process(os.getpid())

for file in glob("test-pages/*/source.html"):
    html = open(file).read()
    json.dumps(parse_html(html, url))
    idx += 1

print(idx, process.memory_info().rss / 1024**2, time.time() - s)
    