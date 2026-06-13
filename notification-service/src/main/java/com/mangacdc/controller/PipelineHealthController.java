package com.mangacdc.controller;

import com.mangacdc.service.PipelineHealthService;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.Map;

@RestController
@RequestMapping("/api/pipeline")
public class PipelineHealthController {

    private final PipelineHealthService pipelineHealthService;

    public PipelineHealthController(PipelineHealthService pipelineHealthService) {
        this.pipelineHealthService = pipelineHealthService;
    }

    @GetMapping("/health")
    public Map<String, Object> health() {
        return pipelineHealthService.buildHealth();
    }
}
