"""
Naver Search Workflow for Alfred 5
Copyright (c) 2024 Inchan Song

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
"""

import sys
import re
from workflow import web, Workflow

def get_data(word):
    url = 'https://opendict.korean.go.kr/search/autoComplete'
    data = {'searchTerm': word}
    r = web.post(url, data)
    r.raise_for_status()
    return r.json()

def parse_suggestions(js_code):
    # 배열 내용을 추출하기 위한 정규 표현식
    pattern = r"var dq_searchResultList=new Array\((.*?)\);"
    match = re.search(pattern, js_code)
    if not match:
        return []
    array_contents = match.group(1)
    # 배열 요소를 분할하고 따옴표 제거
    suggestions = [s.strip().strip("'") for s in array_contents.split(',')]
    return suggestions

def main(wf):
    args = wf.args[0]
    wf.add_item(
        title="'%s' 검색하기" % args,
        autocomplete=args,
        arg=args,
        quicklookurl='https://opendict.korean.go.kr/search/searchResult?focus_name=query&query=%s' % args,
        valid=True
    )

    def wrapper():
        return get_data(args)

    res_json = wf.cached_data('opendict_%s' % args, wrapper, max_age=30)

    # res_json이 없거나 'json' 키가 없을 경우 처리
    if not res_json or 'json' not in res_json or not res_json['json']:
        wf.add_item(
            title=f"'{args}'에 대한 검색 결과가 없습니다",
            icon='noresults.png',
            valid=False
        )
        wf.send_feedback()
        return

    js_code = res_json['json'][0]
    suggestions = parse_suggestions(js_code)

    # suggestions가 비어 있을 경우 처리
    if not suggestions:
        wf.add_item(
            title=f"'{args}'에 대한 검색 결과가 없습니다",
            icon='noresults.png',
            valid=False
        )
    else:
        for suggestion in suggestions:
            wf.add_item(
                title=suggestion,
                autocomplete=suggestion,
                arg=suggestion,
                copytext=suggestion,
                largetext=suggestion,
                quicklookurl='https://opendict.korean.go.kr/search/searchResult?focus_name=query&query=%s' % suggestion,
                valid=True
            )

    wf.send_feedback()

if __name__ == '__main__':
    wf = Workflow()
    sys.exit(wf.run(main))
