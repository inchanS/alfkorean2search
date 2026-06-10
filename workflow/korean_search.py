"""
stdict Korean Search Workflow for Alfred 5
Copyright (c) 2026 Inchan Song
"""

import os
import sys
import re
from typing import List, Dict, Any
from workflow import web, Workflow

def get_data(word: str, api_key: str) -> Dict[str, Any]:
    """
    국립국어원 표준국어대사전 OpenAPI를 호출하여 JSON 응답을 반환합니다.
    """
    url = 'https://stdict.korean.go.kr/api/search.do'
    params = {
        'key': api_key,
        'q': word,
        'req_type': 'json'
    }
    r = web.get(url, params=params)
    r.raise_for_status()
    return r.json()

def parse_suggestions(res_json: Dict[str, Any]) -> List[Dict[str, str]]:
    """
    JSON 응답을 파싱하여 표시할 단어(동음이의어 번호 포함), 뜻풀이, 고유 URL을 반환합니다.
    """
    suggestions = []

    channel = res_json.get('channel', {})
    items = channel.get('item', [])

    # 단일 항목 반환 시 리스트로 변환
    if isinstance(items, dict):
        items = [items]

    for item in items:
        # 1. 단어 정리 (API 상의 불필요한 기호 제거)
        word = item.get('word', '')
        pure_word = word.replace('^', '').replace('-', '')

        # 2. 동음이의어 번호 표기 (예: 사전(3))
        sup_no = item.get('sup_no', '0')
        if str(sup_no) not in ('', '0'):
            display_word = f"{pure_word}({sup_no})"
        else:
            display_word = pure_word

        # 3. 해당 단어의 고유 상세 페이지 URL 추출 및 생성
        target_code = item.get('target_code', '')
        item_link = item.get('link', '')

        if not item_link and target_code:
            item_link = f'https://stdict.korean.go.kr/search/searchView.do?word_no={target_code}&searchKeywordTo=3'

        # 4. 뜻풀이 파싱
        senses = item.get('sense', [])
        if isinstance(senses, dict):
            senses = [senses]

        for sense in senses:
            definition = sense.get('definition', '설명 없음')
            suggestions.append({
                'word': display_word,
                'definition': definition,
                'link': item_link
            })

    return suggestions

def main(wf: Workflow) -> None:
    args = wf.args[0] if len(wf.args) > 0 else ""
    api_key = os.getenv("API_KEY")

    if not api_key:
        wf.add_item(
            title="API 키가 설정되지 않았습니다.",
            subtitle="Alfred 워크플로우의 [Environment Variables]에서 API_KEY를 입력해주세요.",
            valid=False
        )
        wf.send_feedback()
        return

    # [수정됨] 첫 줄: 일반 전체 검색 결과 페이지 주소를 arg에 직접 할당
    general_search_url = f'https://stdict.korean.go.kr/search/searchResult.do?pageSize=10&searchKeyword={args}'
    wf.add_item(
        title="'%s' 전체 검색하기" % args,
        subtitle="표준국어대사전에서 '%s'의 전체 검색 결과를 확인합니다." % args,
        autocomplete=args,
        arg=general_search_url,
        quicklookurl=general_search_url,
        valid=True
    )

    def wrapper() -> Dict[str, Any]:
        return get_data(args, api_key)

    try:
        res_json = wf.cached_data('stdict_%s' % args, wrapper, max_age=3600)
    except Exception as e:
        wf.add_item(
            title="API 요청 중 오류가 발생했습니다. 사전에 없는 검색어일 수 있습니다.",
            subtitle=str(e),
            valid=False
        )
        wf.send_feedback()
        return

    if not res_json or 'channel' not in res_json or 'item' not in res_json['channel']:
        wf.send_feedback()
        return

    suggestions = parse_suggestions(res_json)

    if suggestions:
        # 중복 제거 로직
        seen = set()
        for sug in suggestions:
            word = sug['word']
            definition = sug['definition']
            link = sug['link']

            unique_key = f"{word}_{definition}"
            if unique_key in seen:
                continue
            seen.add(unique_key)

            # [수정됨] 세부 뜻풀이 항목: 특정 단어의 고유 페이지 주소를 arg에 직접 할당
            wf.add_item(
                title=word,
                subtitle=definition,
                autocomplete=word,
                arg=link,
                copytext=word,
                largetext=definition,
                quicklookurl=link,
                valid=True
            )

    wf.send_feedback()

if __name__ == '__main__':
    wf = Workflow()
    sys.exit(wf.run(main))